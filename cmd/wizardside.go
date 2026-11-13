package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/client/macos/stages/host"
)

// wizardSide is what differs between the wizards: their stages, the process that caffeinate waits
// on, the Configure forms, and the state screen's read.
type wizardSide struct {
	stages func(wizardEffects, wizardOptions) []step.Step
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
