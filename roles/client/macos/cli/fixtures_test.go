package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
	ubuntucli "github.com/zadykian/ws/roles/server/ubuntu/cli"
)

const (
	// enterScreen and leaveScreen are how a terminal's alternate screen starts and ends.
	enterScreen = "\x1b[?1049h"
	leaveScreen = "\x1b[?1049l"
)

// fakeProcess returns an Env that touches nothing of the test's own.
func fakeProcess() enginecli.Env { return clitest.Env() }

// withEnv returns p with the environment env, for Getenv, Lookup and Setenv.
func withEnv(p enginecli.Env, env map[string]string) enginecli.Env { return clitest.WithEnv(p, env) }

// screenCommand returns a command that keys type on.
func screenCommand(t *testing.T, keys string) *cobra.Command {
	c := &cobra.Command{Use: "apply"}
	c.SetContext(t.Context())
	c.SetIn(strings.NewReader(keys))
	c.SetErr(&bytes.Buffer{})
	return c
}

// runWs runs args on the root of both groups, with stdin, and returns the status, stdout and stderr.
func runWs(p enginecli.Env, stdin string, args ...string) (status int, stdout, stderr string) {
	return clitest.Run(p, []*cobra.Command{clientGroup(p), serverGroup(p)}, stdin, args...)
}

// clientGroup is ws client, over this package's commands.
func clientGroup(p enginecli.Env) *cobra.Command {
	return enginecli.Group("client", "the client's systems", Command(p))
}

// serverGroup is ws server, over the server's commands.
func serverGroup(p enginecli.Env) *cobra.Command {
	return enginecli.Group("server", "the server's systems", ubuntucli.Command(p))
}
