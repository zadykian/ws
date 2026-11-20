package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"syscall"
	"testing"
	"time"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// stops are the signals that end a run of add or remove as a failure does.
var stops = []syscall.Signal{syscall.SIGINT, syscall.SIGTERM}

// pending is what gh prints of checks that still run, so that the landing waits.
const pending = `[{"name":"values","bucket":"pending"}]`

func TestSecretsAddRemovesTheCloneOnASignal(t *testing.T) {
	for _, sig := range stops {
		p, mac := addFixture(nil)
		mac.Results[checksLine] = effecttest.Result{Out: []byte(pending)}
		p.Machine = &signaled{Mac: mac, sig: sig, at: checksLine}
		status, _, stderr := addWs(p, "", "lab")
		if status != 1 || !strings.Contains(stderr, "stopped waiting for the checks") {
			t.Errorf("add on %v: status = %d, stderr = %q", sig, status, stderr)
		}
		if calls := mac.Calls; calls[len(calls)-1] != "run rm -rf "+mac.Base {
			t.Errorf("add on %v leaves the clone: %q", sig, calls)
		}
	}
}

func TestRemoveRemovesTheCloneOnASignal(t *testing.T) {
	for _, sig := range stops {
		p, mac := removeSecretsFixture(heldBy(t, labKey))
		mac.Results[checksLine] = effecttest.Result{Out: []byte(pending)}
		p.Machine = &signaled{Mac: mac, sig: sig, at: checksLine}
		status, _, stderr := runWs(p, "client", "macos", "remove", "lab")
		if status != 1 || !strings.Contains(stderr, "stopped waiting for the checks") ||
			!strings.Contains(stderr, "lab stays on the Mac") {
			t.Errorf("remove on %v: status = %d, stderr = %q", sig, status, stderr)
		}
		calls := mac.Calls
		if calls[len(calls)-1] != "run rm -rf "+mac.Base || callAt(calls, "ssh-keygen -R") >= 0 {
			t.Errorf("remove on %v leaves the clone, or forgets the host: %q", sig, calls)
		}
	}
}

// A signal at the question on the scanned key ends add, which would otherwise wait for an answer
// that never comes.
func TestSecretsAddStopsAtTheFingerprintOnASignal(t *testing.T) {
	for _, sig := range stops {
		p, mac := addFixture(nil)
		scannable(mac, "lab")
		p.Machine = &signaled{Mac: mac, sig: sig, at: "ssh-keyscan -t ed25519 -p 22022 192.0.2.7"}
		// A terminal where nobody answers.
		in, w := io.Pipe()
		var out, errs bytes.Buffer
		done := make(chan int, 1)
		go func() {
			done <- enginecli.Execute(context.Background(), enginecli.Root(p, clientGroup(p)),
				[]string{"client", "macos", "secrets", "add", "lab"}, in, &out, &errs)
		}()
		select {
		case status := <-done:
			if status != 1 || !strings.Contains(errs.String(), "no answer on the host key of lab") {
				t.Errorf("add on %v: status = %d, stderr = %q", sig, status, errs.String())
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("add on %v still waits for the answer", sig)
		}
		_ = w.Close()
		if _, ok := mac.Machine.Files[knownHosts]; ok || cloned(mac.Calls) {
			t.Errorf("add on %v recorded the key, or cloned the secrets: %q", sig, mac.Calls)
		}
	}
}
