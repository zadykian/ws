package cmd

import (
	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/tui"
)

// opensList says that a run opens the list of steps first: it draws a screen, and neither a
// selection nor --check is given. Either of those goes straight to the run.
func opensList(c *cobra.Command, p cli.Env, selected bool) bool {
	check, _ := c.Flags().GetBool("check")
	return engine.DrawsScreen(cli.CallOf(c, p)) && !selected && !check
}

// pickSteps draws the list of groups over scan and returns the choice. A choice that names no step
// is a maintainer who left.
func pickSteps(c *cobra.Command, p cli.Env, groups []tui.Group, scan tui.Work) (tui.Choice, error) {
	return tui.Pick(c.Context(), engine.ScreenConfig(cli.CallOf(c, p)), c.CommandPath(), groups, scan)
}
