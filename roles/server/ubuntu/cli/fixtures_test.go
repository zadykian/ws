package cli

import (
	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
)

// fakeProcess returns an Env that touches nothing of the test's own.
func fakeProcess() enginecli.Env { return clitest.Env() }

// runWs runs args on the root of ws server, with stdin, and returns the status, stdout and stderr.
func runWs(p enginecli.Env, stdin string, args ...string) (status int, stdout, stderr string) {
	server := enginecli.Group("server", "the server's systems",
		enginecli.Group("ubuntu", "the commands for an Ubuntu server", NewValuesCmd(p), NewSecretsCmd(p)))
	return clitest.Run(p, []*cobra.Command{server}, stdin, args...)
}
