package engine

import (
	"context"
	"io"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/engine/tui"
)

// WizardOptions is what a run of a wizard's stages takes beside its effects: the flags, and what
// the lists chose. Nil Left and Only run every stage whole.
type WizardOptions struct {
	Check, Update bool
	// Only is ws's roles where the lists chose ws's run in part, as --only's entries.
	Only []string
	// Left is the run's LeftOut, which reaches the steps of the Mac's apply too.
	Left func(parent, name string) bool
}

// WizardEffects is what the stages act through: a plain run's terminal, or a screen's Bridge.
type WizardEffects struct {
	// Machine is the machine that the stages act on.
	Machine effect.Machine
	Prog    string
	Talks   effect.Foreground
	Ask     effect.Prompt
	Hidden  effect.Hidden
	Out     effect.Out
	// Text takes what a run prints on stdout.
	Text io.Writer
	Obs  step.Observer
	Hand *wizard.Hand
	// Sudo is the side's way to root on ws, which Side.NewSudo made. A run takes it from the scan
	// before it, so that its password is asked once.
	Sudo any
	// Stop is the run's Stop, which reaches the steps of the Mac's apply too. Nil runs all.
	Stop func() error
	// Events hears ws's run. A plain run has none: it follows ws's run by its lines.
	Events step.Observer
}

// WsRun is the stage of ws's run, as the lists and the scan read it.
type WsRun struct {
	Name string
	// Await is the stage that ws's report waits for.
	Await string
	// State asks ws what its roles hold.
	State func(ctx context.Context) ([]role.RoleState, error)
}

// Side is what differs between the wizards: their stages, the process that caffeinate waits on,
// and what the lists show of them.
type Side struct {
	Stages func(e WizardEffects, o WizardOptions) []step.Step
	// Pid is the process that caffeinate waits on. 0 on ws, which has no caffeinate.
	Pid int
	// Local is the machine of a stage that names none.
	Local wizard.Side
	// NewSudo makes the sudo that a scan, the state screen's read and the run share. Nil makes none.
	NewSudo func() any
	// Read is the state screen's read of side's machine, and what ends it.
	Read func(side string, sudo any) (tui.StateReader, func(context.Context))
	// Nested gives the groups that the stage named runs, as the lists show them under it.
	Nested func(stage string) []tui.Group
	// WsRun finds the stage of ws's run in list.
	WsRun func(list []step.Step) (WsRun, bool)
	// Before builds the steps that run first, on lines, before the lock and the screen.
	Before func(o WizardOptions) []func(effect.Out) step.Step
}
