package cli

import (
	"time"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
)

// fakeProcess returns an Env that touches nothing of the test's own.
func fakeProcess() enginecli.Env { return clitest.Env() }

// runWs runs args on the root of ws server, with stdin, and returns the status, stdout and stderr.
func runWs(p enginecli.Env, stdin string, args ...string) (status int, stdout, stderr string) {
	return clitest.Run(p, []*cobra.Command{serverGroup(p)}, stdin, args...)
}

// serverGroup is ws server, over this package's commands.
func serverGroup(p enginecli.Env) *cobra.Command {
	return enginecli.Group("server", "the server's systems", Command(p))
}

// withEnv returns p with the environment env, for Getenv, Lookup and Setenv.
func withEnv(p enginecli.Env, env map[string]string) enginecli.Env { return clitest.WithEnv(p, env) }

const (
	// enterScreen and leaveScreen are how a terminal's alternate screen starts and ends.
	enterScreen = "\x1b[?1049h"
	leaveScreen = "\x1b[?1049l"
)

// repeating types keys again and again, so that one lands once the list takes it.
type repeating string

func (r repeating) Read(b []byte) (int, error) {
	time.Sleep(5 * time.Millisecond)
	return copy(b, r), nil
}
