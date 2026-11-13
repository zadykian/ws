package cmd

import (
	"context"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/roles/client/macos"
	"github.com/zadykian/ws/roles/client/macos/stages/tools"
	"github.com/zadykian/ws/roles/server/ubuntu"
	"github.com/zadykian/ws/roles/server/ubuntu/stages"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/client/macos/stages/host"
)

// wizardSide is what differs between the wizards: their stages, the process that caffeinate waits
// on, the Configure forms, and the state screen's read.
type wizardSide struct {
	stages func(wizardEffects, engine.WizardOptions) []step.Step
	// pid is the process that caffeinate waits on. 0 on ws, which has no caffeinate.
	pid   int
	forms func(context.Context) ([]tui.Sheet, func(context.Context), error)
	read  func(string, *host.Sudo) (tui.StateReader, func(context.Context))
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
