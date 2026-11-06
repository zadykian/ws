package main

import (
	"context"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/internal/server/apply"
	"github.com/zadykian/ws/internal/server/roles"
	"github.com/zadykian/ws/internal/tui"
)

// opensList says that a run opens the list of steps first: it draws a screen, and neither a
// selection nor --check is given. Either of those goes straight to the run.
func opensList(c *cobra.Command, p process, selected bool) bool {
	check, _ := c.Flags().GetBool("check")
	return drawsScreen(c, p) && !selected && !check
}

// pickSteps draws the list of groups over scan and returns the choice. A choice that names no step
// is a maintainer who left.
func pickSteps(c *cobra.Command, p process, groups []tui.Group, scan tui.Work) (tui.Choice, error) {
	theme := tui.NewTheme(p.getenv)
	cfg := tui.Config{
		In: c.InOrStdin(), Out: c.ErrOrStderr(), Theme: theme,
		Cut: func(s syscall.Signal) error { return signalExit{s} },
	}
	return tui.Pick(c.Context(), cfg, c.CommandPath(), groups, scan)
}

// pickServer opens the list of the roles' steps, scanned by a check, and returns cfg and the roles
// as the choice sets them. The roles are nil where the maintainer left.
func pickServer(c *cobra.Command, p process, cfg apply.Config, list []roles.Role) (apply.Config, []roles.Role, error) {
	scan := func(ctx context.Context, b *tui.Bridge) error {
		look := cfg
		look.Check, look.Only = true, nil
		look.M = screenMachine(ctx, b, cfg.M)
		look.Out, look.Observer, look.Stop, look.Prepare = b, b, b.Stop(ctx), prepareSteps(look.M, b)
		_, err := apply.Run(ctx, look, list)
		return err
	}
	choice, err := pickSteps(c, p, serverGroups(cfg, list)[1:], scan)
	if err != nil || len(choice.Only) == 0 {
		return cfg, nil, err
	}
	cfg.Only, cfg.Check = choice.Only, choice.Check
	chosen, err := apply.Select(list, cfg.Only)
	return cfg, chosen, err
}
