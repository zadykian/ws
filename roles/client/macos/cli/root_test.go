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
		{"no command", []string{"client", "macos"}, 0, "Available Commands:", ""},
		{"help", []string{"client", "macos", "-h"}, 0, "Available Commands:", ""},
		{"unknown", []string{"client", "macos", "nope"}, 2, "", "ws client macos: unknown command \"nope\" for \"ws client macos\"\nRun 'ws client macos --help' for usage.\n"},
		{"tunnel alone", []string{"client", "macos", "tunnel"}, 0, "Available Commands:", ""},
		{"tunnel unknown", []string{"client", "macos", "tunnel", "nope"}, 2, "", "ws client macos tunnel: unknown command"},
		{"tunnel status extra", []string{"client", "macos", "tunnel", "status", "x"}, 2, "", "ws client macos tunnel status: unknown command \"x\""},
		{"tunnel loaded extra", []string{"client", "macos", "tunnel", "loaded", "x"}, 2, "", "ws client macos tunnel loaded: unknown command \"x\""},
		{"tunnel routes alone", []string{"client", "macos", "tunnel", "routes"}, 2, "", "ws client macos tunnel routes: requires at least 1 arg(s), only received 0\nRun 'ws client macos tunnel routes --help' for usage.\n"},
		{"resolver alone", []string{"client", "macos", "resolver"}, 2, "", "ws client macos resolver: requires at least 1 arg(s)"},
		{"resolver flag", []string{"client", "macos", "resolver", "--nope", "x"}, 2, "", "ws client macos resolver: unknown flag: --nope"},
		{"wizard flag", []string{"client", "macos", "wizard", "--bogus"}, 2, "", "ws client macos wizard: unknown flag: --bogus\nRun 'ws client macos wizard --help' for usage.\n"},
		{"wizard help", []string{"client", "macos", "wizard", "-h"}, 0, "--check", ""},
		{"conf two files", []string{"client", "macos", "conf", "a", "b"}, 2, "", "ws client macos conf: accepts at most 1 arg(s), received 2"},
		{"apply an argument", []string{"client", "macos", "apply", "ssh"}, 2, "", "ws client macos apply: unknown command \"ssh\" for \"ws client macos apply\""},
		{"secrets alone", []string{"client", "macos", "secrets"}, 0, "Available Commands:", ""},
		{"secrets unknown", []string{"client", "macos", "secrets", "nope"}, 2, "", "ws client macos secrets: unknown command \"nope\""},
		{"secrets init extra", []string{"client", "macos", "secrets", "init", "x"}, 2, "", "ws client macos secrets init: unknown command \"x\""},
		{"secrets set alone", []string{"client", "macos", "secrets", "set"}, 0, "Available Commands:", ""},
		{"secrets set no name", []string{"client", "macos", "secrets", "set", "env"}, 2, "", "ws client macos secrets set env: accepts 1 arg(s), received 0"},
		{"secrets set extra", []string{"client", "macos", "secrets", "set", "env", "A", "x"}, 2, "", "ws client macos secrets set env: accepts 1 arg(s), received 2"},
		{"secrets set file no name", []string{"client", "macos", "secrets", "set", "file"}, 2, "", "ws client macos secrets set file: accepts between 1 and 2 arg(s), received 0"},
		{"secrets set git flag", []string{"client", "macos", "secrets", "set", "git", "https://git.example.com", "--bogus"}, 2, "", "ws client macos secrets set git: unknown flag: --bogus"},
		{"secrets unset one", []string{"client", "macos", "secrets", "unset", "env"}, 2, "", "ws client macos secrets unset: accepts 2 arg(s), received 1"},
		{"secrets keys extra", []string{"client", "macos", "secrets", "keys", "x"}, 2, "", "ws client macos secrets keys: unknown command \"x\""},
		{"apply unknown step", []string{"client", "macos", "apply", "--only", "nope"}, 2, "", "ws client macos apply: no step is named \"nope\""},
		{"apply no step", []string{"client", "macos", "apply", "--only", ""}, 2, "", "ws client macos apply: --only names no step"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			status, stdout, stderr := runWs(fakeProcess(), "", tc.args...)
			if status != tc.wantStatus || !strings.Contains(stdout, tc.wantOut) || !strings.Contains(stderr, tc.wantErr) {
				t.Errorf("status = %d, stdout = %q, stderr = %q; want %d, %q, %q", status, stdout, stderr, tc.wantStatus, tc.wantOut, tc.wantErr)
			}
		})
	}
}

func TestTunnelStatusAndKeyNeedRoot(t *testing.T) {
	for _, sub := range []string{"status", "key"} {
		status, stdout, stderr := runWs(fakeProcess(), "", "client", "macos", "tunnel", sub)
		want := "ws client macos tunnel " + sub + ": run it as root, with sudo\n"
		if status != 1 || stdout != "" || stderr != want {
			t.Errorf("%s: status = %d, stdout = %q, stderr = %q; want 1, nothing, %q", sub, status, stdout, stderr, want)
		}
	}
}
