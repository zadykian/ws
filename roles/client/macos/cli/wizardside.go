package cli

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/engine"
	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/client/macos"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/stages/host"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
	"github.com/zadykian/ws/roles/client/macos/stages/tools"
	ubuntucli "github.com/zadykian/ws/roles/server/ubuntu/cli"
)

// macSide is ws client macos wizard's side.
func macSide(p enginecli.Env, server *string) engine.Side {
	return engine.Side{
		Stages: func(e engine.WizardEffects, o engine.WizardOptions) []step.Step {
			return wizardStages(effectsOf(p, e), o, *server)
		},
		Pid: p.Pid, Change: macChange, Local: stage.Mac, NewSudo: func() any { return &host.Sudo{} }, Nested: nestedRows, WsRun: ubuntucli.WsRunSide(p),
		Read: func(side string, sudo any) (tui.StateReader, func(context.Context)) {
			return macState(p, server, side, sudo.(*host.Sudo))
		},
		Before: func(o engine.WizardOptions, ask effect.Prompt) []func(effect.Out) step.Step {
			var before []func(effect.Out) step.Step
			if o.Update {
				// Its own run, before the lock: the new ws takes the place of this process, which would leave the
				// lock held. It runs before the screen too, on lines, or the new ws would find a terminal left raw.
				before = append(before, func(out effect.Out) step.Step {
					return &tools.MacUpdate{M: p.Machine, Out: out, Version: p.Version, URL: p.Releases,
						Home: p.Getenv("HOME"), Args: p.Argv}
				})
			}
			if *server == "" {
				// After the update, which runs this command again: the new ws asks, not this one. On lines too,
				// before the lists, which name the server's file.
				before = append(before, func(effect.Out) step.Step { return &askServer{ask: ask, tty: p.TTY, name: server} })
			}
			return before
		},
	}
}

// wizardServer returns the server that the wizard sets up: the one that flag names, or the only one.
// With none and no flag it returns "", and the wizard asks the name. It refuses a second server.
func wizardServer(p enginecli.Env, flag string) (string, error) {
	mac := conf.Path(p.Getenv, p.Pwd)
	name, err := conf.Pick(p.Machine, mac, flag)
	if err == nil {
		return name, nil
	}
	names, listErr := conf.Servers(p.Machine, mac)
	switch {
	case listErr != nil:
		return "", listErr
	case len(names) == 0 && flag == "":
		return "", nil
	case len(names) == 0 && conf.IsName(flag):
		return flag, nil
	case len(names) > 0 && conf.IsName(flag):
		return "", fmt.Errorf("%s would be a second server, beside %s: the wizard sets up one server", flag, conf.List(names))
	}
	return "", err
}

// effectsOf is e as the stages of p take it: p acts on e's machine.
func effectsOf(p enginecli.Env, e engine.WizardEffects) wizardEffects {
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
