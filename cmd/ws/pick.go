package main

import (
	"context"
	"io"
	"slices"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/internal/server/apply"
	"github.com/zadykian/ws/internal/server/roles"
	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/step"
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
	return tui.Pick(c.Context(), screenConfig(c, p), c.CommandPath(), groups, scan)
}

// pickServer opens the list of the roles' steps, scanned by a check, and returns cfg and the roles
// as the choice sets them. The roles are nil where the maintainer left.
func pickServer(c *cobra.Command, p process, cfg apply.Config, list []roles.Role) (apply.Config, []roles.Role, error) {
	p.machine = cfg.M // ws_home confines it.
	scan := func(ctx context.Context, b *tui.Bridge) error {
		look := onScreen(cfg, b)
		look.Check, look.Only = true, nil
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

// pickSetup opens the list of ws mac apply's steps, scanned by a check. It returns the options that
// the choice sets, and false where the maintainer left.
func pickSetup(c *cobra.Command, p process, build func(check bool) setupBuild) (step.Options, bool, error) {
	scan := func(ctx context.Context, b *tui.Bridge) error {
		return runSetup(ctx, build(true)(b.Effects().M, b, io.Discard), b.Options(step.Options{Check: true}))
	}
	rows := tui.NewGroup("", build(true)(p.machine, nil, io.Discard))
	choice, err := pickSteps(c, p, []tui.Group{rows}, scan)
	if err != nil || len(choice.Only) == 0 {
		return step.Options{}, false, err
	}
	// ssh again has no row, as a check has no second plan, so it follows ssh by its wait.
	left := func(_, name string) bool {
		return slices.Contains(rows.Steps, name) && !slices.Contains(choice.Only, name)
	}
	return step.Options{Check: choice.Check, LeftOut: left}, true, nil
}
