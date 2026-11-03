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
	return process{machine: effecttest.NewMachine(), getenv: noEnv, version: "v1.2.3"}
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
	for _, group := range []*cobra.Command{newRoot(p), newServerCmd(p)} {
		walk(group, func(c *cobra.Command) {
			if c.Args == nil {
				t.Errorf("%s: Args is nil", c.CommandPath())
			}
		})
	}
}

func TestTunnelStatusNeedsRoot(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if got := tunnelStatus(501, &stdout, &stderr); got != 1 {
		t.Errorf("status = %d, want 1", got)
	}
	if !strings.Contains(stderr.String(), "run it as root, with sudo") || stdout.Len() != 0 {
		t.Errorf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}

func TestTunnelKeyNeedsRoot(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if got := tunnelKey(501, &stdout, &stderr); got != 1 {
		t.Errorf("status = %d, want 1", got)
	}
	if !strings.Contains(stderr.String(), "run it as root, with sudo") || stdout.Len() != 0 {
		t.Errorf("stdout = %q, stderr = %q", stdout.String(), stderr.String())
	}
}
