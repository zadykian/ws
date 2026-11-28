package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// failingExec is a Machine whose Exec fails, as with no seward at its path.
type failingExec struct{ *effecttest.Machine }

func (failingExec) Exec(string, []string, []string) error { return errors.New("not found") }

// execs returns the exec calls of m.
func execs(m *effecttest.Machine) []string {
	var got []string
	for _, call := range m.Calls {
		if strings.HasPrefix(call, "exec ") {
			got = append(got, call)
		}
	}
	return got
}

// sshKeys runs ws macos ssh with args on a fresh Machine, and returns the status, stderr and the
// commands that replaced ws.
func sshKeys(euid int, args ...string) (status int, stderr string, ran []string) {
	p := fakeProcess()
	p.Euid = euid
	status, _, stderr = runWs(p, append([]string{"macos", "ssh"}, args...)...)
	return status, stderr, execs(p.Machine.(*effecttest.Machine))
}

func TestSSHListHandsSewardTheCall(t *testing.T) {
	status, stderr, ran := sshKeys(501, "list")
	if want := "exec /usr/local/bin/seward list"; status != 0 || len(ran) != 1 || ran[0] != want {
		t.Errorf("status = %d, stderr = %q, execs = %q; want 0 and %q", status, stderr, ran, want)
	}
}

func TestSSHListTakesNoArgument(t *testing.T) {
	if status, _, ran := sshKeys(501, "list", "a"); status != 2 || len(ran) != 0 {
		t.Errorf("status = %d, execs = %q; want 2 and no run", status, ran)
	}
}

func TestSSHKeysRefuseRoot(t *testing.T) {
	status, stderr, ran := sshKeys(0, "list")
	if status != 1 || len(ran) != 0 || !strings.Contains(stderr, "not root") {
		t.Errorf("status = %d, stderr = %q, execs = %q; want 1, the refusal and no run", status, stderr, ran)
	}
}

func TestSSHKeysNameSetupWhereSewardCannotRun(t *testing.T) {
	p := fakeProcess()
	p.Machine = failingExec{effecttest.NewMachine()}
	status, _, stderr := runWs(p, "macos", "ssh", "list")
	if want := "ws macos setup"; status != 1 || !strings.Contains(stderr, "/usr/local/bin/seward") || !strings.Contains(stderr, want) {
		t.Errorf("status = %d, stderr = %q; want 1, the path and %q", status, stderr, want)
	}
}
