package cli

import (
	"context"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/roles/secrets"
	"github.com/zadykian/ws/macos/roles/secrets/secretstest"
)

// checksLine reads the checks of the pull request that the fake repository opens.
const checksLine = secrets.GH + " pr checks " + secretstest.Pull + " --json name,bucket"

// signaled is a Mac that sends ws the signal once it runs the line at, such as the read of checks
// that are still running. Its commands refuse to start once their context ends, as exec's do.
type signaled struct {
	*secretstest.Mac
	sig  syscall.Signal
	at   string
	sent bool
}

func (s *signaled) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out, err := s.Mac.Run(ctx, name, args...)
	if line := strings.Join(append([]string{name}, args...), " "); line == s.at && !s.sent {
		s.sent = true
		if err := syscall.Kill(os.Getpid(), s.sig); err != nil {
			return nil, err
		}
	}
	return out, err
}

func TestSecretsInitRemovesTheCloneOnASignal(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM} {
		m := effecttest.NewMachine()
		m.Files["/usr/local/bin/fnox"], m.Files[secrets.Plugin] = []byte("fnox"), []byte("plugin")
		m.Dirs[secrets.Tree+"/ubuntu/roles"] = 0o755
		m.Results[secrets.Plugin+" --list"] = effecttest.Result{Out: []byte("# Serial: 1, Slot: 1\nage1yubikey1qa\n\n")}
		m.Results[secrets.Plugin+" --identity --slot 1"] = effecttest.Result{Out: []byte("AGE-PLUGIN-YUBIKEY-1QQQ\n")}
		m.Results[secrets.GH+" pr list --repo o/n --state open --limit 100 --json headRefName,url --jq "+
			".[] | [.headRefName, .url] | @tsv"] = effecttest.Result{}
		m.Results[checksLine] = effecttest.Result{Out: []byte(`[{"name":"values","bucket":"pending"}]`)}
		mac := secretstest.New(m, nil)
		p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
		p.Machine = &signaled{Mac: mac, sig: sig, at: checksLine}
		status, _, stderr := clitest.Run(p, []*cobra.Command{Command(p)}, "\n", "macos", "secrets", "init")
		if status != 1 || !strings.Contains(stderr, "stopped waiting for the checks") {
			t.Errorf("init on %v: status = %d, stderr = %q", sig, status, stderr)
		}
		if calls := mac.Calls; calls[len(calls)-1] != "run rm -rf "+mac.Base {
			t.Errorf("init on %v leaves the clone: %q", sig, calls)
		}
	}
}

func TestSecretsInitRefusesRoot(t *testing.T) {
	p := fakeProcess()
	p.Euid = 0
	status, _, stderr := runWs(p, "macos", "secrets", "init")
	if status != 1 || !strings.Contains(stderr, "run it as yourself, not root") {
		t.Errorf("init as root: status = %d, stderr = %q", status, stderr)
	}
	if calls := p.Machine.(*effecttest.Machine).Calls; len(calls) != 0 {
		t.Errorf("init as root ran %q", calls)
	}
}

func TestSecretsInitTakesNoServer(t *testing.T) {
	status, _, stderr := runWs(fakeProcess(), "macos", "secrets", "init", "--server", "ws")
	if status != 1 || !strings.Contains(stderr, "init takes no --server") {
		t.Errorf("init --server ws: status = %d, stderr = %q", status, stderr)
	}
}

func TestSecretsInitTakesNoArgument(t *testing.T) {
	status, _, stderr := runWs(fakeProcess(), "macos", "secrets", "init", "ws")
	if status != 2 || !strings.Contains(stderr, `unknown command "ws"`) {
		t.Errorf("init ws: status = %d, stderr = %q", status, stderr)
	}
}

func TestSecretsInitReadsTheTree(t *testing.T) {
	status, _, stderr := runWs(fakeProcess(), "macos", "secrets", "init")
	if status != 1 || !strings.Contains(stderr, "/opt/ws/mise.toml") {
		t.Errorf("init without a tree: status = %d, stderr = %q", status, stderr)
	}
}
