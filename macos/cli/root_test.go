package cli

import (
	"strings"
	"testing"
)

func TestMacHelpAndErrors(t *testing.T) {
	for _, tc := range []struct {
		name       string
		args       []string
		wantStatus int
		wantOut    string
		wantErr    string
	}{
		{"no command", []string{"macos"}, 0, "Available Commands:", ""},
		{"help", []string{"macos", "-h"}, 0, "Available Commands:", ""},
		{"unknown", []string{"macos", "nope"}, 2, "", "ws macos: unknown command \"nope\" for \"ws macos\"\nRun 'ws macos --help' for usage.\n"},
		{"tunnel alone", []string{"macos", "tunnel"}, 0, "Available Commands:", ""},
		{"tunnel unknown", []string{"macos", "tunnel", "nope"}, 2, "", "ws macos tunnel: unknown command"},
		{"tunnel status extra", []string{"macos", "tunnel", "status", "x"}, 2, "", "ws macos tunnel status: unknown command \"x\""},
		{"tunnel loaded extra", []string{"macos", "tunnel", "loaded", "x"}, 2, "", "ws macos tunnel loaded: unknown command \"x\""},
		{"tunnel routes alone", []string{"macos", "tunnel", "routes"}, 2, "", "ws macos tunnel routes: requires at least 1 arg(s), only received 0\nRun 'ws macos tunnel routes --help' for usage.\n"},
		{"resolver alone", []string{"macos", "resolver"}, 2, "", "ws macos resolver: requires at least 1 arg(s)"},
		{"resolver flag", []string{"macos", "resolver", "--nope", "x"}, 2, "", "ws macos resolver: unknown flag: --nope"},
		{"values print two files", []string{"macos", "values", "print", "a", "b"}, 2, "", "ws macos values print: accepts at most 1 arg(s), received 2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, stdout, stderr := runWs(fakeProcess(), tc.args...)
			if status != tc.wantStatus || !strings.Contains(stdout, tc.wantOut) || !strings.Contains(stderr, tc.wantErr) {
				t.Errorf("status = %d, stdout = %q, stderr = %q; want %d, %q, %q", status, stdout, stderr, tc.wantStatus, tc.wantOut, tc.wantErr)
			}
		})
	}
}

func TestTunnelStatusAndKeyNeedRoot(t *testing.T) {
	for _, sub := range []string{"status", "key"} {
		status, stdout, stderr := runWs(fakeProcess(), "macos", "tunnel", sub)
		want := "ws macos tunnel " + sub + ": run it as root, with sudo\n"
		if status != 1 || stdout != "" || stderr != want {
			t.Errorf("%s: status = %d, stdout = %q, stderr = %q; want 1, nothing, %q", sub, status, stdout, stderr, want)
		}
	}
}
