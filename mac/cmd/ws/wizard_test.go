package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

func testWizardEnv(goos string, uid int, stderr *bytes.Buffer) wizardEnv {
	env := map[string]string{"TMPDIR": "/t/"}
	return wizardEnv{goos: goos, uid: uid, pid: 7, m: effecttest.NewMachine(),
		getenv: func(k string) string { return env[k] }, stdin: strings.NewReader(""), stderr: stderr}
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
	if got := wizardCommand(context.Background(), []string{"--check"}, e); got != 0 || stderr.Len() != 0 {
		t.Errorf("status = %d, stderr = %q; want a silent run", got, stderr.String())
	}
	if calls := e.m.(*effecttest.Machine).Calls; len(calls) == 0 || calls[0] != "mkdir /t/ws-wizard.lock 700" {
		t.Errorf("calls = %v; want the lock first", calls)
	}
}
