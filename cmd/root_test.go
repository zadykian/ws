package cmd

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
)

// fakeProcess returns an Env that touches nothing of the test's own.
func fakeProcess() cli.Env { return clitest.Env() }

// runWs runs args on the root of both groups, with stdin, and returns the status, stdout and stderr.
func runWs(p cli.Env, stdin string, args ...string) (status int, stdout, stderr string) {
	return clitest.Run(p, []*cobra.Command{newMacCmd(p), newServerCmd(p)}, stdin, args...)
}

// runGroups is runWs on the groups of this system.
func runGroups(p cli.Env, args ...string) (status int, stdout, stderr string) {
	return clitest.Run(p, groups(p), "", args...)
}

func TestEveryCommandSetsArgs(t *testing.T) {
	p := fakeProcess()
	for _, group := range []*cobra.Command{cli.Root(p), newMacCmd(p), newServerCmd(p)} {
		cli.Walk(group, func(c *cobra.Command) {
			if c.Args == nil {
				t.Errorf("%s: Args is nil", c.CommandPath())
			}
		})
	}
}
