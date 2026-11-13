package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/role"
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

// pickServer opens the list of the roles' steps, scanned by a check, and returns cfg and the roles
// as the choice sets them. The roles are nil where the maintainer left.
func pickServer(c *cobra.Command, p cli.Env, cfg role.Config, list []role.Role) (role.Config, []role.Role, error) {
	p.Machine = cfg.M // ws_home confines it.
	scan := func(ctx context.Context, b *tui.Bridge) error {
		look := onScreen(cfg, b)
		look.Check, look.Only = true, nil
		_, err := role.Run(ctx, look, list)
		return err
	}
	choice, err := pickSteps(c, p, serverGroups(cfg, list)[1:], scan)
	if err != nil || len(choice.Only) == 0 {
		return cfg, nil, err
	}
	cfg.Only, cfg.Check = choice.Only, choice.Check
	chosen, err := role.Select(list, cfg.Only)
	return cfg, chosen, err
}
