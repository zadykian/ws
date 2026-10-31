package main

import (
	"bytes"
	"strings"
	"testing"
)

func testWizardEnv(goos string, uid int, stderr *bytes.Buffer) wizardEnv {
	return wizardEnv{goos: goos, uid: uid, getenv: func(string) string { return "" }, stderr: stderr}
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
		got := wizardCommand(tc.args, testWizardEnv(tc.goos, tc.uid, &stderr))
		if got != tc.status || !strings.Contains(stderr.String(), tc.message) {
			t.Errorf("%s: status = %d, stderr = %q; want %d, %q", name, got, stderr.String(), tc.status, tc.message)
		}
	}
}
