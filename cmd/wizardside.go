package cmd

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/internal/wizard"
)

// wizardSide is what differs between the wizards: their stages, the process that caffeinate waits
// on, the Configure forms, and the state screen's read.
type wizardSide struct {
	stages func(wizardEffects, wizardOptions) []step.Step
	// pid is the process that caffeinate waits on. 0 on ws, which has no caffeinate.
	pid   int
	forms func(context.Context) ([]tui.Sheet, func(context.Context), error)
	read  func(string, *wizard.Sudo) (tui.StateReader, func(context.Context))
}

// macSide is ws client macos wizard's side.
func macSide(c *cobra.Command, p process) wizardSide {
	return wizardSide{
		stages: wizardStages,
		pid:    p.pid,
		forms:  macForms(c, p),
		read: func(side string, sudo *wizard.Sudo) (tui.StateReader, func(context.Context)) {
			return macState(p, side, sudo)
		},
	}
}

// serverSide is ws server ubuntu wizard's side. Every stage runs on ws, with no caffeinate, and the
// forms are ws's values alone.
func serverSide(p process) wizardSide {
	return wizardSide{
		stages: serverWizardStages,
		forms: func(context.Context) ([]tui.Sheet, func(context.Context), error) {
			return serverSheets(p), nil, nil
		},
		read: func(string, *wizard.Sudo) (tui.StateReader, func(context.Context)) {
			return stateRead(serverOpen, p.version), nil
		},
	}
}
