package cmd

import (
	"context"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/client/macos"
	"github.com/zadykian/ws/roles/client/macos/stages/host"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
	"github.com/zadykian/ws/roles/client/macos/stages/tools"
	ubuntucli "github.com/zadykian/ws/roles/server/ubuntu/cli"
)

// macSide is ws client macos wizard's side.
func macSide(p cli.Env) engine.Side {
	return engine.Side{
		Stages: func(e engine.WizardEffects, o engine.WizardOptions) []step.Step {
			return wizardStages(effectsOf(p, e), o)
		},
		Pid: p.Pid, Change: macChange, Local: stage.Mac, NewSudo: func() any { return &host.Sudo{} }, Nested: nestedRows, WsRun: ubuntucli.WsRunSide(p),
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

// effectsOf is e as the stages of p take it: p acts on e's machine.
func effectsOf(p cli.Env, e engine.WizardEffects) wizardEffects {
	p.Machine = e.Machine
	sudo, _ := e.Sudo.(*host.Sudo)
	return wizardEffects{p: p, prog: e.Prog, talks: e.Talks, ask: e.Ask, hidden: e.Hidden, out: e.Out, text: e.Text,
		obs: e.Obs, hand: e.Hand, sudo: sudo, stop: e.Stop, events: e.Events}
}

// nestedRows returns the groups that the stage named runs, as the lists show them under it.
func nestedRows(stage string) []tui.Group {
	if stage == (&tools.Setup{}).Name() {
		return []tui.Group{{Name: stage, Steps: macos.Names()}}
	}
	return ubuntucli.NestedRows(stage)
}
