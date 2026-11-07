package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

// After a full run of this ws, from the Mac or from ws, with nothing changed, no apply runs.
func TestServerWizardAfterARunAppliesNothing(t *testing.T) {
	p, m := wizardProcess("---\nsecurity_outbound: true\nws_user: ada\n", true,
		"run v1.2.3 0 on -\nactive -\nuser ada\noutbound on\nsecrets -\nclaude ok\n")
	status, stdout, stderr := runWs(p, "", "server", "wizard", "--plain")
	if status != 0 || stdout != "" || slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, "-- run") }) {
		t.Errorf("status %d, stdout %q, stderr %q; calls %q", status, stdout, stderr, m.Calls)
	}
}

// The values are asked at a terminal and written, and a container's run is skipped with a line.
func TestServerWizardAsksTheValuesAndSkipsUnbooted(t *testing.T) {
	p, m := wizardProcess("", false, "")
	p.tty = true
	m.Results["env SUDO_USER=root sh -s -- status"] = effecttest.Result{Out: []byte("claude ok\n")}
	status, _, stderr := runWs(p, "cloud\n\n", "server", "wizard", "--plain")
	if status != 0 || !strings.Contains(stderr, "isn't booted") {
		t.Fatalf("status %d, stderr %q", status, stderr)
	}
	if got := string(m.Files[serverVars]); got != "---\nsecurity_outbound: false\nws_user: root\n" {
		t.Errorf("host_vars = %q", got)
	}
}

// Under --check the values are named, not asked, and ws's run waits on them. A user is refused.
func TestServerWizardCheckAsksNothing(t *testing.T) {
	p, m := wizardProcess("", true, "")
	p.euid = 1000
	if status, _, stderr := runWs(p, "", "server", "wizard", "--check", "--plain"); status == 0 || !strings.Contains(stderr, "by sudo") {
		t.Errorf("as a user: status %d, stderr %q", status, stderr)
	}
	p.euid = 0
	status, _, stderr := runWs(p, "", "server", "wizard", "--check", "--plain")
	if status != 0 || !strings.Contains(stderr, "would ask security_outbound and ws_user") || len(m.Files[serverVars]) != 0 {
		t.Errorf("status %d, stderr %q, host_vars %q", status, stderr, m.Files[serverVars])
	}
}

// At a terminal the stages run on the run screen, which ends on the normal screen.
func TestServerWizardAtATerminalDrawsTheRunScreen(t *testing.T) {
	p, _ := wizardProcess("---\nsecurity_outbound: true\nws_user: ada\n", false, "claude ok\n")
	p = withEnv(p, map[string]string{"TERM": "xterm"})
	p.tty, p.errTTY = true, true
	status, stdout, stderr := runWs(p, "", "server", "wizard")
	if status != 0 || stdout != "" || !strings.Contains(stderr, enterScreen) || !strings.Contains(stderr, leaveScreen) {
		t.Errorf("status %d, stdout %q, stderr %q", status, stdout, stderr)
	}
}
