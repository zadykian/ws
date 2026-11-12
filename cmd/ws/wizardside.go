package main

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/step"
)

// wizardSide is what differs between the wizards: their stages, the process that caffeinate waits
// on, the Configure forms, and the state screen's read.
type wizardSide struct {
	stages func(wizardEffects, wizardOptions) []step.Step
	// pid is the process that caffeinate waits on. 0 on ws, which has no caffeinate.
	pid   int
	forms func(context.Context) ([]tui.Sheet, func(context.Context), error)
	read  func(tui.Side, *wizard.Sudo) (tui.StateReader, func(context.Context))
}

// macSide is ws mac wizard's side.
func macSide(c *cobra.Command, p process) wizardSide {
	return wizardSide{
		stages: wizardStages,
		pid:    p.pid,
		forms:  macForms(c, p),
		read: func(side tui.Side, sudo *wizard.Sudo) (tui.StateReader, func(context.Context)) {
			return macState(p, side, sudo)
		},
	}
}
