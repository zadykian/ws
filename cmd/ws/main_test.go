package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/effect/effecttest"
)

// fakeProcess returns a process that touches nothing of the test's own.
func fakeProcess() process {
	return process{
		machine: effecttest.NewMachine(), getenv: noEnv, lookup: func(string) (string, bool) { return "", false },
		setenv: func(string, string) error { return nil }, euid: 501, pid: 7, goos: "darwin", pwd: "/p", version: "v1.2.3",
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
//
//nolint:unparam // the commands of ws mac secrets read stdin, in the units that port them
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
	status := &cobra.Command{Use: "status", Args: cobra.NoArgs, RunE: func(*cobra.Command, []string) error {
		return exitStatus(3)
	}}
	var out, errs bytes.Buffer
	root := newRoot(fakeProcess(), fail, status)
	if got := execute(context.Background(), root, []string{"fail"}, strings.NewReader(""), &out, &errs); got != 1 ||
		errs.String() != "ws fail: one\nws fail: two\n" {
		t.Errorf("fail: status = %d, stderr = %q", got, errs.String())
	}
	errs.Reset()
	if got := execute(context.Background(), root, []string{"status"}, strings.NewReader(""), &out, &errs); got != 3 || errs.Len() != 0 {
		t.Errorf("status: status = %d, stderr = %q", got, errs.String())
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
		{"no command", []string{"mac"}, 0, "Available Commands:", ""},
		{"help", []string{"mac", "-h"}, 0, "Available Commands:", ""},
		{"unknown", []string{"mac", "nope"}, 2, "", "ws mac: unknown command \"nope\" for \"ws mac\"\nRun 'ws mac --help' for usage.\n"},
		{"tunnel alone", []string{"mac", "tunnel"}, 0, "Available Commands:", ""},
		{"tunnel unknown", []string{"mac", "tunnel", "nope"}, 2, "", "ws mac tunnel: unknown command"},
		{"tunnel status extra", []string{"mac", "tunnel", "status", "x"}, 2, "", "ws mac tunnel status: unknown command \"x\""},
		{"tunnel loaded extra", []string{"mac", "tunnel", "loaded", "x"}, 2, "", "ws mac tunnel loaded: unknown command \"x\""},
		{"tunnel routes alone", []string{"mac", "tunnel", "routes"}, 2, "", "ws mac tunnel routes: requires at least 1 arg(s), only received 0\nRun 'ws mac tunnel routes --help' for usage.\n"},
		{"resolver alone", []string{"mac", "resolver"}, 2, "", "ws mac resolver: requires at least 1 arg(s)"},
		{"resolver flag", []string{"mac", "resolver", "--nope", "x"}, 2, "", "ws mac resolver: unknown flag: --nope"},
		{"wizard flag", []string{"mac", "wizard", "--bogus"}, 2, "", "ws mac wizard: unknown flag: --bogus\nRun 'ws mac wizard --help' for usage.\n"},
		{"wizard help", []string{"mac", "wizard", "-h"}, 0, "--check", ""},
		{"conf two files", []string{"mac", "conf", "a", "b"}, 2, "", "ws mac conf: accepts at most 1 arg(s), received 2"},
		{"apply two steps", []string{"mac", "apply", "ssh", "git"}, 2, "", "ws mac apply: accepts at most 1 arg(s), received 2"},
		{"apply unknown step", []string{"mac", "apply", "nope"}, 2, "", "ws mac apply: invalid argument \"nope\" for \"ws mac apply\""},
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
		status, stdout, stderr := runWs(fakeProcess(), "", "mac", "tunnel", sub)
		want := "ws mac tunnel " + sub + ": run it as root, with sudo\n"
		if status != 1 || stdout != "" || stderr != want {
			t.Errorf("%s: status = %d, stdout = %q, stderr = %q; want 1, nothing, %q", sub, status, stdout, stderr, want)
		}
	}
}
