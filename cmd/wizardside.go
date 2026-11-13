package cmd

import (
	"context"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/client/macos"
	"github.com/zadykian/ws/roles/client/macos/stages/host"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
	"github.com/zadykian/ws/roles/client/macos/stages/tools"
	"github.com/zadykian/ws/roles/server/ubuntu"
	"github.com/zadykian/ws/roles/server/ubuntu/stages"
)

// macSide is ws client macos wizard's side.
func macSide(p cli.Env) engine.Side {
	return engine.Side{
		Stages: func(e engine.WizardEffects, o engine.WizardOptions) []step.Step {
			return wizardStages(effectsOf(p, e), o)
		},
		Pid: p.Pid, Change: macChange, Local: stage.Mac, NewSudo: func() any { return &host.Sudo{} }, Nested: nestedRows, WsRun: wsRunSide(p),
		Read: func(side string, sudo any) (tui.StateReader, func(context.Context)) {
			return macState(p, side, sudo.(*host.Sudo))
		},
		Before: func(o engine.WizardOptions) []func(effect.Out) step.Step {
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

// serverSide is ws server ubuntu wizard's side. Every stage runs on ws, with no caffeinate.
func serverSide(p cli.Env) engine.Side {
	return engine.Side{
		Stages: func(e engine.WizardEffects, o engine.WizardOptions) []step.Step {
			return serverWizardStages(effectsOf(p, e), o)
		},
		Local: stage.Mac, Change: macChange, Nested: nestedRows, WsRun: wsRunSide(p),
		Read: func(string, any) (tui.StateReader, func(context.Context)) {
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

// wsRunOf returns the stage of ws's run, or nil where the list holds none.
func wsRunOf(list []step.Step) *stages.WsRun {
	for _, s := range list {
		if run, ok := s.(*stages.WsRun); ok {
			return run
		}
	}
	return nil
}
