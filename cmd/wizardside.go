package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/client/macos"
	"github.com/zadykian/ws/roles/client/macos/stages/host"
	"github.com/zadykian/ws/roles/client/macos/stages/tools"
	"github.com/zadykian/ws/roles/server/ubuntu"
	"github.com/zadykian/ws/roles/server/ubuntu/stages"
)

// wizardSide is what differs between the wizards: their stages, the process that caffeinate waits
// on, the Configure forms, and the state screen's read.
type wizardSide struct {
	stages func(wizardEffects, engine.WizardOptions) []step.Step
	// pid is the process that caffeinate waits on. 0 on ws, which has no caffeinate.
	pid   int
	forms func(context.Context) ([]tui.Sheet, func(context.Context), error)
	read  func(string, *host.Sudo) (tui.StateReader, func(context.Context))
	// before builds the steps that run first, on lines, before the lock and the screen.
	before func(o engine.WizardOptions) []func(effect.Out) step.Step
}

// macSide is ws client macos wizard's side.
func macSide(c *cobra.Command, p cli.Env) wizardSide {
	return wizardSide{
		stages: wizardStages,
		pid:    p.Pid,
		forms:  macForms(c.InOrStdin(), c.ErrOrStderr(), p),
		read: func(side string, sudo *host.Sudo) (tui.StateReader, func(context.Context)) {
			return macState(p, side, sudo)
		},
		before: func(o engine.WizardOptions) []func(effect.Out) step.Step {
			if !o.Update {
				return nil
			}
			// Its own run, before the lock: the new ws takes the place of this process, which would leave the
			// lock held. It runs before the screen too, on lines, or the new ws would find a terminal left raw.
			return []func(effect.Out) step.Step{func(out effect.Out) step.Step {
				return &tools.MacUpdate{M: p.Machine, Out: out, Version: p.Version, URL: p.Releases,
					Home: p.Getenv("HOME"), Args: p.Argv}
			}}
		},
	}
}

// serverSide is ws server ubuntu wizard's side. Every stage runs on ws, with no caffeinate, and the
// forms are ws's values alone.
func serverSide(p cli.Env) wizardSide {
	return wizardSide{
		stages: serverWizardStages,
		forms: func(context.Context) ([]tui.Sheet, func(context.Context), error) {
			return serverSheets(p), nil, nil
		},
		read: func(string, *host.Sudo) (tui.StateReader, func(context.Context)) {
			return stateRead(serverOpen, p.Version), nil
		},
	}
}

// effectsOf is e as the stages of p take it: p acts on e's machine.
func effectsOf(p cli.Env, e engine.WizardEffects) wizardEffects {
	p.Machine = e.Machine
	sudo, _ := e.Sudo.(*host.Sudo)
	return wizardEffects{p: p, prog: e.Prog, talks: e.Talks, ask: e.Ask, hidden: e.Hidden, out: e.Out, text: e.Text,
		obs: e.Obs, hand: e.Hand, sudo: sudo, stop: e.Stop, events: e.Events}
}

// wsRunSide finds the stage of ws's run in a list, with what the lists ask of it.
func wsRunSide(p cli.Env) func([]step.Step) (engine.WsRun, bool) {
	return func(list []step.Step) (engine.WsRun, bool) {
		run := wsRunOf(list)
		if run == nil {
			return engine.WsRun{}, false
		}
		return engine.WsRun{Name: run.Name(), Await: run.WaitsOn(), State: func(ctx context.Context) ([]role.RoleState, error) {
			return run.Ws.State(ctx, p.Version)
		}}, true
	}
}

// nestedRows returns the groups that the stage named runs, as the lists show them under it.
func nestedRows(stage string) []tui.Group {
	var groups []tui.Group
	switch stage {
	case (&tools.Setup{}).Name():
		groups = []tui.Group{{Name: stage, Steps: macos.Names()}}
	case (&stages.WsRun{}).Name():
		for _, name := range ubuntu.Names() {
			groups = append(groups, tui.Group{Name: name})
		}
	}
	return groups
}
