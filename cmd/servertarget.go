package cmd

import (
	"context"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
)

// serverTarget is ws server ubuntu apply's side of the verbs. opts holds the flags, read when the
// command runs.
func serverTarget(p cli.Env, opts *applyOptions) engine.Target {
	return engine.Target{Prepare: func(_ context.Context, c engine.Call) (engine.Run, error) {
		r, err := prepareServer(c, p, *opts)
		if err != nil {
			return engine.Run{}, err
		}
		r.term.Flush() // Prepare's lines wait for no run: Apply says the run's own.
		return runOf(r.cfg, r.list), nil
	}, State: func(_ context.Context, c engine.Call) (engine.Report, error) {
		return serverReport(c, p, *opts)
	}}
}

// serverReport readies what ws server ubuntu state reports.
func serverReport(c engine.Call, p cli.Env, opts applyOptions) (engine.Report, error) {
	r, err := prepareServer(c, p, opts)
	if err != nil {
		return engine.Report{}, err
	}
	// State reads the machine and takes no secrets. A step that needs them would write them out to
	// plan, as ikeward's install does, and a report has no use for them.
	list, err := r.roles(secrets.Secrets{Unread: true})
	if err != nil {
		return engine.Report{}, engine.UsageError{Err: err}
	}
	return engine.Report{
		M: r.cfg.M,
		Read: func(ctx context.Context, b *tui.Bridge) ([]tui.StateSection, error) {
			// The read takes the screen's effects as a run does, as the steps don't own the terminal under it.
			states, err := role.State(ctx, onScreen(r.cfg, b), list)
			return engine.RoleSections(states), err
		},
		Print: func(ctx context.Context, c engine.Call) error {
			states, err := role.State(ctx, r.cfg, list)
			r.term.Flush()
			if err != nil {
				return err
			}
			if c.JSON {
				return engine.WriteJSON(c.Out, role.StateDoc{Version: p.Version, Roles: states})
			}
			return engine.WriteRoles(c.Out, states)
		},
	}, nil
}

// runOf is the run of the roles in list under cfg's options.
func runOf(cfg role.Config, list []role.Role) engine.Run {
	run := engine.Run{M: cfg.M, Listed: 1}
	run.Groups = func(check bool) []tui.Group {
		cfg.Check = check
		groups := []tui.Group{tui.NewGroup(role.Prepare, cfg.Prepare)}
		for _, r := range list {
			groups = append(groups, tui.NewGroup(r.Name, cfg.Steps(r)))
		}
		return groups
	}
	run.Do = func(ctx context.Context, e engine.Effects, o step.Options) (string, error) {
		at := cfg
		at.M, at.Out, at.Observer, at.Stop = e.M, e.Out, o.Observer, o.Stop
		at.Check, at.Prepare = o.Check, prepareSteps(e.M, e.Out)
		changed, err := role.Run(ctx, at, list)
		if err != nil {
			return "", err
		}
		return role.Summary(changed, o.Check), nil
	}
	run.Choose = func(choice tui.Choice) (engine.Run, step.Options, error) {
		cfg.Only = choice.Only
		chosen, err := role.Select(list, cfg.Only)
		return runOf(cfg, chosen), step.Options{Check: choice.Check}, err
	}
	return run
}
