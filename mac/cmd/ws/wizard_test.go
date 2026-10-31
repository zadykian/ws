package main

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

func testWizardEnv(goos string, uid int, stderr *bytes.Buffer) wizardEnv {
	env := map[string]string{"TMPDIR": "/t/"}
	return wizardEnv{goos: goos, uid: uid, pid: 7, m: effecttest.NewMachine(),
		getenv: func(k string) string { return env[k] },
		lookup: func(k string) (string, bool) { v, ok := env[k]; return v, ok }, pwd: "/p", stdin: strings.NewReader(""), stderr: stderr}
}

func TestWizardCommandRefuses(t *testing.T) {
	for name, tc := range map[string]struct {
		args        []string
		goos        string
		uid, status int
		message     string
	}{
		"help":     {[]string{"--help"}, "darwin", 501, 0, "usage: ws wizard [--check] [--update]"},
		"argument": {[]string{"--now"}, "darwin", 501, 1, "ws wizard: unknown argument '--now'; usage: ws wizard"},
		"linux":    {nil, "linux", 501, 1, "ws wizard: it sets up a Mac, not Linux"},
		"root":     {nil, "darwin", 0, 1, "run it as yourself, not root: it asks sudo where it needs root"},
	} {
		var stderr bytes.Buffer
		got := wizardCommand(context.Background(), tc.args, testWizardEnv(tc.goos, tc.uid, &stderr))
		if got != tc.status || !strings.Contains(stderr.String(), tc.message) {
			t.Errorf("%s: status = %d, stderr = %q; want %d, %q", name, got, stderr.String(), tc.status, tc.message)
		}
	}
}

func TestWizardCommandRuns(t *testing.T) {
	var stderr bytes.Buffer
	e := testWizardEnv("darwin", 501, &stderr)
	if got := wizardCommand(context.Background(), []string{"--check"}, e); got != 1 ||
		!strings.Contains(stderr.String(), "Answers: no terminal to ask WS_TARGET: set WS_TARGET") {
		t.Errorf("status = %d, stderr = %q; want the refusal to ask with no terminal", got, stderr.String())
	}
	if calls := e.m.(*effecttest.Machine).Calls; len(calls) == 0 || calls[0] != "mkdir /t/ws-wizard.lock 700" {
		t.Errorf("calls = %v; want the lock first", calls)
	}
}

func TestWizardCheckShowsTheAnswersDiff(t *testing.T) {
	var stderr bytes.Buffer
	e := testWizardEnv("darwin", 501, &stderr)
	env := map[string]string{"WS_TARGET": "cloud", "WS_REMOTE": "r.example", "WS_REMOTE_PORT": "22", "WS_GIT_NAME": "",
		"WS_TUNNEL_NETWORKS": "", "WS_RESOLVERS": "", "WS_HOOK": ""}
	e.lookup = func(k string) (string, bool) { v, ok := env[k]; return v, ok }
	if got := wizardCommand(context.Background(), []string{"--check"}, e); got != 0 || !strings.Contains(stderr.String(), "+WS_TARGET=cloud") {
		t.Errorf("status = %d, stderr = %q; want the diff of mac.conf", got, stderr.String())
	}
	if got := string(e.m.(*effecttest.Machine).Files["/t/ws-wizard.lock/mac.conf"]); !strings.HasPrefix(got, "WS_TARGET=cloud\n") {
		t.Errorf("scratch mac.conf = %q", got)
	}
}

func TestIsTerminalIsNotDevNull(t *testing.T) {
	null, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = null.Close() }()
	if isTerminal(null) {
		t.Error("isTerminal(/dev/null) = true")
	}
}
