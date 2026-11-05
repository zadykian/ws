package main

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

// macTerminal returns a process with a terminal, as the Mac's user, with a mac.conf and what ssh's
// step looks at.
func macTerminal() (process, *effecttest.Machine) {
	m := effecttest.NewMachine()
	m.Files["/h/.config/ws/mac.conf"] = []byte("WS_USER=ada\n")
	m.Results["mktemp -d"] = effecttest.Result{Out: []byte("/tmp/x\n")}
	m.Results["rm -rf /tmp/x"] = effecttest.Result{}
	for _, file := range []string{"new", "whole"} {
		m.Results["ssh -G -F /tmp/x/"+file+" ws"] = effecttest.Result{}
	}
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h", "TERM": "xterm"})
	p.machine, p.tty, p.errTTY = m, true, true
	return p, m
}

func TestMacApplyAtATerminalDrawsAWholeScreen(t *testing.T) {
	p, _ := macTerminal()
	status, stdout, stderr := runWs(p, "", "mac", "apply", "--check", "ssh")
	if status != 0 || stdout != "" || !strings.Contains(stderr, enterScreen) || !strings.Contains(stderr, leaveScreen) {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

func TestMacApplyReportsAFailurePastTheScreen(t *testing.T) {
	p, m := macTerminal()
	delete(m.Results, "mktemp -d")
	status, _, stderr := runWs(p, "", "mac", "apply", "--check", "ssh")
	left := strings.LastIndex(stderr, leaveScreen)
	if status != 1 || left < 0 || !strings.Contains(stderr[left:], "ws mac apply: ") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}

func TestMacApplyDrawsNothingWithPlain(t *testing.T) {
	p, _ := macTerminal()
	_, wantOut, wantErr := runWs(withEnv(p, map[string]string{"HOME": "/h", "TERM": ""}), "", "mac", "apply", "--check", "ssh")
	status, stdout, stderr := runWs(p, "", "mac", "apply", "--plain", "--check", "ssh")
	if status != 0 || stdout != wantOut || stderr != wantErr || strings.Contains(stderr, "\x1b") {
		t.Errorf("status = %d, stdout = %q, stderr = %q, want the lines %q", status, stdout, stderr, wantErr)
	}
}
