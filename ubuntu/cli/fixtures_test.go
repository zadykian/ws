package cli

import (
	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
)

// fakeProcess returns an Env that touches nothing of the test's own.
func fakeProcess() enginecli.Env { return clitest.Env() }

// runWs runs args on the root of ws ubuntu guest, with stdin, and returns the status, stdout and stderr.
func runWs(p enginecli.Env, stdin string, args ...string) (status int, stdout, stderr string) {
	return clitest.Run(p, []*cobra.Command{Command(p)}, stdin, args...)
}
