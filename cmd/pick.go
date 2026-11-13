package cmd

import (
	"context"
	"io"
	"slices"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
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

// pickSetup opens the list of ws client macos apply's steps, scanned by a check. It returns the
// options that the choice sets, and false where the maintainer left.
func pickSetup(c *cobra.Command, p cli.Env, rows tui.Group, run setupRun) (step.Options, bool, error) {
	scan := func(ctx context.Context, b *tui.Bridge) error {
		return run(ctx, b.Effects().M, b, io.Discard, b.Options(step.Options{Check: true}))
	}
	choice, err := pickSteps(c, p, []tui.Group{rows}, scan)
	if err != nil || len(choice.Only) == 0 {
		return step.Options{}, false, err
	}
	left := func(_, name string) bool { return !slices.Contains(choice.Only, name) }
	return step.Options{Check: choice.Check, LeftOut: left}, true, nil
}
