package main

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

const (
	// enterScreen and leaveScreen are how a terminal's alternate screen starts and ends.
	enterScreen = "\x1b[?1049h"
	leaveScreen = "\x1b[?1049l"
	// dpkgCheck is the look at a package that a run's Prepare makes first.
	dpkgCheck = "dpkg-query -W -f ${Package} ${db:Status-Abbrev}\n ca-certificates"
)

// serverTerminal returns a process with a terminal, as root on the one system that ws server apply
// takes, with ca-certificates installed.
func serverTerminal() (process, *effecttest.Machine) {
	p := withEnv(fakeProcess(), map[string]string{"TERM": "xterm"})
	p.tty, p.errTTY, p.euid = true, true, 0
	m := p.machine.(*effecttest.Machine)
	m.Files["/etc/os-release"] = []byte("ID=ubuntu\nVERSION_ID=\"26.04\"\n")
	m.Results[dpkgCheck] = effecttest.Result{Out: []byte("ca-certificates ii \n")}
	return p, m
}

func TestServerApplyAtATerminalEndsOnTheNormalScreen(t *testing.T) {
	p, m := serverTerminal()
	m.Results["dpkg-query -W -f ${Package} ${db:Status-Abbrev}\n tmux"] = effecttest.Result{Out: []byte("tmux ii \n")}
	status, stdout, stderr := runWs(p, "", "server", "apply", "--check", "--only", "tmux")
	left := strings.LastIndex(stderr, leaveScreen)
	if status != 0 || stdout != "" || !strings.Contains(stderr, enterScreen) || left < 0 {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	if want := "ws server apply: would change: none\n"; !strings.HasSuffix(stderr[left:], want) {
		t.Errorf("after the screen: %q, want the last line %q", stderr[left:], want)
	}
}

func TestServerApplyReportsAFailurePastTheScreen(t *testing.T) {
	p, m := serverTerminal()
	delete(m.Results, dpkgCheck)
	status, _, drawn := runWs(p, "", "server", "apply", "--only", "tmux")
	left := strings.LastIndex(drawn, leaveScreen)
	if status != 1 || left < 0 || !strings.Contains(drawn[left:], "ws server apply: Packages: ") {
		t.Errorf("status = %d, stderr = %q", status, drawn)
	}
}
