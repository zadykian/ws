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
		{"no command", []string{"client"}, 0, "Available Commands:", ""},
		{"help", []string{"client", "-h"}, 0, "Available Commands:", ""},
		{"unknown", []string{"client", "nope"}, 2, "", "ws client: unknown command \"nope\" for \"ws client\"\nRun 'ws client --help' for usage.\n"},
		{"tunnel alone", []string{"client", "tunnel"}, 0, "Available Commands:", ""},
		{"tunnel unknown", []string{"client", "tunnel", "nope"}, 2, "", "ws client tunnel: unknown command"},
		{"tunnel status extra", []string{"client", "tunnel", "status", "x"}, 2, "", "ws client tunnel status: unknown command \"x\""},
		{"tunnel loaded extra", []string{"client", "tunnel", "loaded", "x"}, 2, "", "ws client tunnel loaded: unknown command \"x\""},
		{"tunnel routes alone", []string{"client", "tunnel", "routes"}, 2, "", "ws client tunnel routes: requires at least 1 arg(s), only received 0\nRun 'ws client tunnel routes --help' for usage.\n"},
		{"resolver alone", []string{"client", "resolver"}, 2, "", "ws client resolver: requires at least 1 arg(s)"},
		{"resolver flag", []string{"client", "resolver", "--nope", "x"}, 2, "", "ws client resolver: unknown flag: --nope"},
		{"values print two files", []string{"client", "values", "print", "a", "b"}, 2, "", "ws client values print: accepts at most 1 arg(s), received 2"},
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
		status, stdout, stderr := runWs(fakeProcess(), "client", "tunnel", sub)
		want := "ws client tunnel " + sub + ": run it as root, with sudo\n"
		if status != 1 || stdout != "" || stderr != want {
			t.Errorf("%s: status = %d, stdout = %q, stderr = %q; want 1, nothing, %q", sub, status, stdout, stderr, want)
		}
	}
}
