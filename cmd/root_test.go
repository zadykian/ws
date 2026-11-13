package cmd

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"syscall"
	"testing"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// fakeProcess returns a process that touches nothing of the test's own.
func fakeProcess() process {
	return process{
		machine: effecttest.NewMachine(), getenv: noEnv, lookup: func(string) (string, bool) { return "", false },
		setenv: func(string, string) error { return nil }, euid: 501, pid: 7, goos: "darwin", goarch: "arm64", pwd: "/p", version: "v1.2.3",
		self: func() (string, error) { return "/h/.local/bin/ws", nil },
	}
}

// withEnv returns p with the environment env, for getenv, lookup and setenv.
func withEnv(p process, env map[string]string) process {
	p.getenv = func(k string) string { return env[k] }
	p.lookup = func(k string) (string, bool) { v, ok := env[k]; return v, ok }
	p.setenv = func(k, v string) error { env[k] = v; return nil }
	return p
}

// runWs runs args on the root of the groups that p serves, with stdin, and returns the status,
// stdout and stderr.
func runWs(p process, stdin string, args ...string) (status int, stdout, stderr string) {
	var out, errs bytes.Buffer
	root := newRoot(p, newMacCmd(p), newServerCmd(p))
	status = execute(context.Background(), root, args, strings.NewReader(stdin), &out, &errs)
	return status, out.String(), errs.String()
}

// runGroups is runWs on the groups of this system.
func runGroups(p process, args ...string) (status int, stdout, stderr string) {
	var out, errs bytes.Buffer
	status = execute(context.Background(), newRoot(p, groups(p)...), args, strings.NewReader(""), &out, &errs)
	return status, out.String(), errs.String()
}

func TestExecuteExitsAsAShellReportsASignal(t *testing.T) {
	root := newRoot(fakeProcess())
	root.AddCommand(&cobra.Command{Use: "cut", RunE: func(*cobra.Command, []string) error {
		return signalExit{syscall.SIGTERM}
	}})
	var out, errs bytes.Buffer
	status := execute(context.Background(), root, []string{"cut"}, strings.NewReader(""), &out, &errs)
	if status != 143 || errs.Len() != 0 {
		t.Errorf("status = %d, stderr = %q; want 143 and nothing printed", status, errs.String())
	}
}

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStatus int
		wantOut    string
		wantErr    string
	}{
		{"no command", nil, 0, "Available Commands:", ""},
		{"help", []string{"help"}, 0, "Available Commands:", ""},
		{"flag", []string{"-h"}, 0, "Available Commands:", ""},
		{"version", []string{"version"}, 0, "v1.2.3\n", ""},
		{"unknown", []string{"nope"}, 2, "", "ws: unknown command \"nope\" for \"ws\"\nRun 'ws --help' for usage.\n"},
		{"unknown flag", []string{"--nope"}, 2, "", "ws: unknown flag: --nope\nRun 'ws --help' for usage.\n"},
		{"version takes no argument", []string{"version", "x"}, 2, "", "ws version: unknown command \"x\" for \"ws version\"\nRun 'ws version --help' for usage.\n"},
		{"update takes no argument", []string{"update", "x"}, 2, "", "ws update: unknown command \"x\" for \"ws update\"\nRun 'ws update --help' for usage.\n"},
		{"old setup", []string{"setup", "ssh"}, 2, "", `unknown command "setup"`},
		{"old tunnel", []string{"tunnel", "status"}, 2, "", `unknown command "tunnel"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, stdout, stderr := runWs(fakeProcess(), "", tt.args...)
			if status != tt.wantStatus {
				t.Errorf("status = %d, want %d", status, tt.wantStatus)
			}
			if !strings.Contains(stdout, tt.wantOut) {
				t.Errorf("stdout = %q, want it to contain %q", stdout, tt.wantOut)
			}
			if !strings.Contains(stderr, tt.wantErr) {
				t.Errorf("stderr = %q, want it to contain %q", stderr, tt.wantErr)
			}
		})
	}
}

func TestExecuteFailure(t *testing.T) {
	fail := &cobra.Command{Use: "fail", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error {
		return errors.Join(errors.New("one"), errors.New("two"))
	}}
	var out, errs bytes.Buffer
	root := newRoot(fakeProcess(), fail)
	if got := execute(context.Background(), root, []string{"fail"}, strings.NewReader(""), &out, &errs); got != 1 ||
		errs.String() != "ws fail: one\nws fail: two\n" {
		t.Errorf("fail: status = %d, stderr = %q", got, errs.String())
	}
}

func TestEveryCommandSetsArgs(t *testing.T) {
	p := fakeProcess()
	for _, group := range []*cobra.Command{newRoot(p), newMacCmd(p), newServerCmd(p)} {
		walk(group, func(c *cobra.Command) {
			if c.Args == nil {
				t.Errorf("%s: Args is nil", c.CommandPath())
			}
		})
	}
}

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
