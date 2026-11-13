// Package clitest builds the Env and runs the root that the tests of ws's commands share.
package clitest

import (
	"bytes"
	"context"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// Env returns an Env that touches nothing of the test's own.
func Env() cli.Env {
	return cli.Env{
		Machine: effecttest.NewMachine(), Getenv: func(string) string { return "" },
		Lookup: func(string) (string, bool) { return "", false },
		Setenv: func(string, string) error { return nil }, Euid: 501, Pid: 7, GOOS: "darwin", GOARCH: "arm64", Pwd: "/p", Version: "v1.2.3",
		Self: func() (string, error) { return "/h/.local/bin/ws", nil },
	}
}

// WithEnv returns e with the environment env, for Getenv, Lookup and Setenv.
func WithEnv(e cli.Env, env map[string]string) cli.Env {
	e.Getenv = func(k string) string { return env[k] }
	e.Lookup = func(k string) (string, bool) { v, ok := env[k]; return v, ok }
	e.Setenv = func(k, v string) error { env[k] = v; return nil }
	return e
}

// Run runs args on the root of the groups, with stdin, and returns the status, stdout and stderr.
func Run(e cli.Env, groups []*cobra.Command, stdin string, args ...string) (status int, stdout, stderr string) {
	var out, errs bytes.Buffer
	status = cli.Execute(context.Background(), cli.Root(e, groups...), args, strings.NewReader(stdin), &out, &errs)
	return status, out.String(), errs.String()
}
