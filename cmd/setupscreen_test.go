package cmd

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// keyed gives the Mac a mac.conf that names the file key it has for ws's login, as ssh's step needs.
func keyed(m *effecttest.Machine) {
	m.Files["/h/.config/ws/mac.conf"] = []byte("WS_USER=ada\nWS_IDENTITY_PUB_KEY=~/.ssh/id_ed25519.pub\n")
	m.Files["/h/.ssh/id_ed25519.pub"] = []byte("ssh-ed25519 AAAA ada\n")
	m.Results["ssh-keygen -l -f /h/.ssh/id_ed25519.pub"] = effecttest.Result{}
}

// macTerminal returns a process with a terminal, as the Mac's user, with a mac.conf and what ssh's
// step looks at.
func macTerminal() (cli.Env, *effecttest.Machine) {
	m := effecttest.NewMachine()
	keyed(m)
	m.Results["mktemp -d"] = effecttest.Result{Out: []byte("/tmp/x\n")}
	m.Results["rm -rf /tmp/x"] = effecttest.Result{}
	for _, file := range []string{"new", "whole"} {
		m.Results["ssh -G -F /tmp/x/"+file+" ws"] = effecttest.Result{}
	}
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h", "TERM": "xterm"})
	p.Machine, p.TTY, p.ErrTTY = m, true, true
	return p, m
}

func TestMacApplyAtATerminalDrawsAWholeScreen(t *testing.T) {
	p, _ := macTerminal()
	status, stdout, stderr := runWs(p, "", "client", "macos", "apply", "--check", "--only", "ssh")
	if status != 0 || stdout != "" || !strings.Contains(stderr, enterScreen) || !strings.Contains(stderr, leaveScreen) {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

func TestMacApplyReportsAFailurePastTheScreen(t *testing.T) {
	p, m := macTerminal()
	delete(m.Results, "mktemp -d")
	status, _, stderr := runWs(p, "", "client", "macos", "apply", "--check", "--only", "ssh")
	left := strings.LastIndex(stderr, leaveScreen)
	if status != 1 || left < 0 || !strings.Contains(stderr[left:], "ws client macos apply: ") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}

func TestMacApplyDrawsNothingWithPlain(t *testing.T) {
	p, _ := macTerminal()
	_, wantOut, wantErr := runWs(withEnv(p, map[string]string{"HOME": "/h", "TERM": ""}), "", "client", "macos", "apply", "--check", "--only", "ssh")
	status, stdout, stderr := runWs(p, "", "client", "macos", "apply", "--plain", "--check", "--only", "ssh")
	if status != 0 || stdout != wantOut || stderr != wantErr || strings.Contains(stderr, "\x1b") {
		t.Errorf("status = %d, stdout = %q, stderr = %q, want the lines %q", status, stdout, stderr, wantErr)
	}
}
