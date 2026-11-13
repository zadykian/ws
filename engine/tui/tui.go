// Package tui runs ws's full-screen terminal UI: the entry points that the commands call, and the
// types that they hand to them. The screens are in internal packages.
package tui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui/internal/configure"
	"github.com/zadykian/ws/engine/tui/internal/list"
	"github.com/zadykian/ws/engine/tui/internal/pick"
	"github.com/zadykian/ws/engine/tui/internal/queue"
	"github.com/zadykian/ws/engine/tui/internal/run"
	"github.com/zadykian/ws/engine/tui/internal/state"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// A Theme styles every part of a screen.
type Theme = widget.Theme

// NewTheme returns the theme for an environment, which getenv reads.
func NewTheme(getenv func(string) string) Theme { return widget.NewTheme(getenv) }

// A FieldSpec describes one field of a sheet.
type FieldSpec = widget.FieldSpec

// The kinds of a sheet's field.
const (
	TextField     = widget.TextField
	PasswordField = widget.PasswordField
	ChoiceField   = widget.ChoiceField
)

// A Group is the steps of a role or a stage, by the names that a run's events give them.
type Group = list.Group

// NewGroup names steps as a run of them under parent will.
func NewGroup(parent string, steps []step.Step) Group { return list.NewGroup(parent, steps) }

// Config is what Run takes of the process.
type Config = run.Config

// Work is what a run does. It takes b as its effects, and sends the screen its messages by them.
type Work = run.Work

// Bridge joins a work to its screen: the work reports by it, and the machine it runs on is its.
type Bridge = run.Bridge

// StateSection is a part of what a machine reports: a role's steps, or the Mac's, which have no name.
type StateSection = run.StateSection

// RunScreen follows a run: its steps by role with the current one marked, a window on what the run
// prints, a bar of the steps ended and the elapsed time.
type RunScreen = run.Screen

// Run draws screen on c.Out, runs work in a goroutine beside it, and returns once the terminal is
// back.
func Run(ctx context.Context, c Config, screen tea.Model, work Work) error {
	return run.Run(ctx, c, screen, work)
}

// Offscreen returns a Bridge with no screen, whose works run on m as it stands. A test runs a work
// by it.
func Offscreen(m effect.Machine) *Bridge { return run.Offscreen(m) }

// NewRunScreen returns the screen of a run of groups, which name every step before the run starts.
func NewRunScreen(title string, theme Theme, groups ...Group) RunScreen {
	return run.NewRunScreen(title, theme, groups...)
}

// Sheet is a file that the Configure screen edits.
type Sheet = configure.Sheet

// Configure draws the sheets on c.Out, a form each, until a leave, with title over them. It returns
// nil past a leave, and Cut's error past Ctrl-C or a signal, which drop what is unsaved. Saves run
// here, off the screen's goroutine, and what they print stays on the screen.
func Configure(ctx context.Context, c Config, title string, sheets ...Sheet) error {
	return configure.Configure(ctx, c, title, sheets...)
}

// A Choice is what a flat list leaves to a run. No steps says that nothing runs.
type Choice = pick.Choice

// Pick draws the list of groups over scan, a run with --check, and returns what the maintainer
// chose: a Choice with no steps where they left. Ctrl-C ends it as Cut says, and no stop is asked.
func Pick(ctx context.Context, c Config, title string, groups []Group, scan Work) (Choice, error) {
	return pick.Pick(ctx, c, title, groups, scan)
}

// Stage is a step of a queue. The lists show one order, as stages wait on each other across the
// machines and share a connection, so that no two run at once.
type Stage = queue.Stage

// QueueChoice is what the queue's screen leaves to the wizard's loop.
type QueueChoice = queue.Choice

// Next is what the queue's screen was left for.
type Next = queue.Next

// The ways to leave the queue's screen.
const (
	NextLeave     = queue.NextLeave
	NextRun       = queue.NextRun
	NextCheck     = queue.NextCheck
	NextConfigure = queue.NextConfigure
	NextState     = queue.NextState
)

// QueueRun follows a run of a queue on the lists that chose it, over what the run prints.
type QueueRun = queue.RunScreen

// NewQueueRun returns the screen of a run of choice's stages.
func NewQueueRun(title string, theme Theme, choice QueueChoice) QueueRun {
	return queue.NewRunScreen(title, theme, choice)
}

// PickStages draws a queue's lists over scan, as Pick draws a list. A QueueChoice that runs nothing,
// or goes to the form or the state screen, says it by its Next.
func PickStages(ctx context.Context, c Config, title string, stages []Stage, scan Work) (QueueChoice, error) {
	return queue.PickStages(ctx, c, title, stages, scan)
}

// Reopen draws the lists again as left, a NextState choice, found them, and returns the next choice.
// No scan runs.
func Reopen(ctx context.Context, c Config, left QueueChoice) (QueueChoice, error) {
	return queue.Reopen(ctx, c, left)
}

// StateReader reads a machine and changes nothing. b gives the read's effects, for its lines to
// reach the screen and not the terminal under it.
type StateReader = state.StateReader

// RunState draws the state screen on c.Out until q, and reads the machine in Run's work. So a
// read's panic and a signal end as a run's do. A read goes on under the screen, which takes the
// keys meanwhile, and q ends it through ctx. It returns nil after q, and Cut's error after a cut.
func RunState(ctx context.Context, c Config, title string, read StateReader) error {
	return state.RunState(ctx, c, title, read)
}
