package main

import (
	"bytes"
	"context"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/effect/effecttest"
)

// After a full run of this ws, from the Mac or from ws, with nothing changed, no apply runs.
func TestServerWizardAfterARunAppliesNothing(t *testing.T) {
	p, m := wizardProcess("---\nsecurity_outbound: true\nws_user: ada\n", true,
		"run v1.2.3 0 on -\nactive -\nuser ada\noutbound on\nsecrets -\nclaude ok\nsigning none\n")
	status, stdout, stderr := runWs(p, "", "server", "wizard", "--plain")
	if status != 0 || stdout != "" || slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, "-- run") }) {
		t.Errorf("status %d, stdout %q, stderr %q; calls %q", status, stdout, stderr, m.Calls)
	}
}

// The values are asked at a terminal and written, and a container's run is skipped with a line.
func TestServerWizardAsksTheValuesAndSkipsUnbooted(t *testing.T) {
	p, m := wizardProcess("", false, "")
	p.tty = true
	m.Results["env SUDO_USER=root sh -s -- status"] = effecttest.Result{Out: []byte("claude ok\nsigning none\n")}
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

// At a terminal the stages open as ws's list alone, and q leaves it with nothing run.
func TestServerWizardAtATerminalOpensOneList(t *testing.T) {
	p, m := wizardProcess("---\nsecurity_outbound: true\nws_user: ada\n", false, "claude ok\nsigning none\n")
	p = withEnv(p, map[string]string{"TERM": "xterm"})
	p.tty, p.errTTY = true, true
	status, stdout, stderr := runWs(p, "q", "server", "wizard")
	if status != 0 || stdout != "" || !strings.Contains(stderr, enterScreen) || !strings.Contains(stderr, leaveScreen) {
		t.Errorf("status %d, stdout %q, stderr %q", status, stdout, stderr)
	}
	if strings.Contains(stderr, "Mac") || strings.Contains(stderr, "configure") || slices.ContainsFunc(m.Calls, ranWs) {
		t.Errorf("stderr %q, calls %q", stderr, m.Calls)
	}
}

// ranWs tells whether a call ran ws's run.
func ranWs(call string) bool { return strings.HasSuffix(call, "sh -s -- run") }

// screenServer is ws server wizard at a terminal, on a booted machine whose remote script reports a
// good last run, and the keys that the lists read, past the scan.
func screenServer(keys io.ReadWriter) (int, *effecttest.Machine) {
	p, m := wizardProcess("---\nsecurity_outbound: true\nws_user: ada\n", true,
		"run v1.2.3 0 on -\nactive -\nuser ada\noutbound on\nsecrets -\nclaude ok\nsigning none\n")
	p = withEnv(p, map[string]string{"TERM": "xterm"})
	p.tty, p.errTTY, p.machine = true, true, freed{m}
	var out bytes.Buffer
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return execute(ctx, newRoot(p, newServerCmd(p)), []string{"server", "wizard"}, keys, &out, keys), m
}

// The stages chosen run on the list's run screen, ws's run left out.
func TestServerWizardRunsTheStagesChosen(t *testing.T) {
	keys := newAfterScan("n \r")
	status, m := screenServer(keys)
	if status != 0 || bytes.Count(keys.drawn.Bytes(), []byte(enterScreen)) != 2 {
		t.Fatalf("status %d, want a scan and a run: calls %q", status, m.Calls)
	}
	if slices.ContainsFunc(m.Calls, ranWs) {
		t.Errorf("ws's run ran, left out: calls %q", m.Calls)
	}
}

// Where GitHub lacks ws's signing key, the list at the end names it, and each device's key.
func TestServerWizardListsTheSigningKeyByHand(t *testing.T) {
	p, m := wizardProcess("---\nsecurity_outbound: true\nws_user: ada\n", true,
		"run v1.2.3 0 on -\nactive -\nuser ada\noutbound on\nsecrets -\nclaude ok\nsigning ssh-ed25519 AAA\n")
	m.Results["runuser -l ada -c gh api user --jq .login 2>/dev/null || git config --global github.user"] =
		effecttest.Result{}
	status, stdout, stderr := runWs(p, "", "server", "wizard", "--plain")
	for _, want := range []string{"What stays by hand", "settings/ssh/new", "ssh-ed25519 AAA", "ws's lease", "ssh-import-id gh:LOGIN"} {
		if status != 0 || !strings.Contains(stdout, want) {
			t.Errorf("status %d, stdout %q lacks %q; stderr %q", status, stdout, want, stderr)
		}
	}
}
