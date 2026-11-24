package cmd

import (
	"testing"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
)

// fakeProcess returns an Env that touches nothing of the test's own.
func fakeProcess() cli.Env { return clitest.Env() }

// updateCmd returns ws update as the server has it.
func updateCmd(p cli.Env) *cobra.Command { return cli.Update(p, "server", true) }

// runGroups runs args on the root of this system's groups. It returns the status, stdout and
// stderr.
func runGroups(p cli.Env, args ...string) (status int, stdout, stderr string) {
	return clitest.Run(p, groups(p), "", args...)
}

func TestEveryCommandSetsArgs(t *testing.T) {
	p := fakeProcess()
	for _, group := range []*cobra.Command{cli.Root(p), macosCmd(p), serverCmd(p), unitCmd(p)} {
		cli.Walk(group, func(c *cobra.Command) {
			if c.Args == nil {
				t.Errorf("%s: Args is nil", c.CommandPath())
			}
		})
	}
}

// TestNoApplyOrState checks that mise's bootstrap and its plan stand alone: no command of either
// side reports the machine, or applies its units outside mise. Only ws update replaces ws, and
// claude's update, a unit's command, updates cld.
func TestNoApplyOrState(t *testing.T) {
	p := fakeProcess()
	cli.Walk(cli.Root(p, macosCmd(p), serverCmd(p), unitCmd(p), updateCmd(p)), func(c *cobra.Command) {
		gone := c.Name() == "apply" || c.Name() == "state" || c.Name() == "update"
		if gone && c.CommandPath() != "ws server claude update" && c.CommandPath() != "ws update" {
			t.Errorf("%s is a command", c.CommandPath())
		}
	})
}
