package cli_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"syscall"
	"testing"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
)

func runWs(p cli.Env, args ...string) (status int, stdout, stderr string) {
	return clitest.Run(p, nil, "", args...)
}

func TestExecuteExitsAsAShellReportsASignal(t *testing.T) {
	root := cli.Root(clitest.Env())
	root.AddCommand(&cobra.Command{Use: "cut", RunE: func(*cobra.Command, []string) error {
		return engine.SignalExit{Sig: syscall.SIGTERM}
	}})
	var out, errs bytes.Buffer
	status := cli.Execute(context.Background(), root, []string{"cut"}, strings.NewReader(""), &out, &errs)
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
			status, stdout, stderr := runWs(clitest.Env(), tt.args...)
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
	root := cli.Root(clitest.Env(), fail)
	if got := cli.Execute(context.Background(), root, []string{"fail"}, strings.NewReader(""), &out, &errs); got != 1 ||
		errs.String() != "ws fail: one\nws fail: two\n" {
		t.Errorf("fail: status = %d, stderr = %q", got, errs.String())
	}
}
