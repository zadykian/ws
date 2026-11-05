package tui

import (
	"charm.land/lipgloss/v2"

	"github.com/zadykian/ws/step"
)

// Group is the steps of a role or a stage, by the names that a run's events give them.
type Group struct {
	// Name is the run's Options.Parent. An empty one draws no heading.
	Name  string
	Steps []string
}

// NewGroup names steps as a run of them under parent will, so that a screen knows each one before
// the run starts.
func NewGroup(parent string, steps []step.Step) Group {
	g := Group{Name: parent}
	for _, s := range steps {
		g.Steps = append(g.Steps, s.Name())
	}
	return g
}

// stepState says how a run left a step.
type stepState int

const (
	stepOpen stepState = iota
	stepSame
	stepChanged
	stepLeftOut
	stepFailed
	stepSkipped
)

// stateAfter is the state that e leaves its step in. Planned opens a step that has ended, as the
// wizard runs its Setup's steps more than once.
func stateAfter(e step.Event) stepState {
	switch e.Kind {
	case step.Done:
		switch {
		case e.LeftOut:
			return stepLeftOut
		case len(e.Changes) != 0:
			return stepChanged
		}
		return stepSame
	case step.Failed:
		return stepFailed
	case step.Skipped:
		return stepSkipped
	case step.Planned, step.Started:
	}
	return stepOpen
}

// look is the mark and the style of a state. The marks differ by shape, as NO_COLOR takes the rest.
func (s stepState) look(theme Theme) (string, lipgloss.Style) {
	switch s {
	case stepSame:
		return "✓", theme.Good
	case stepChanged:
		return "●", theme.Warn
	case stepLeftOut:
		return "○", theme.Muted
	case stepFailed:
		return "✗", theme.Bad
	case stepSkipped:
		return "–", theme.Muted
	case stepOpen:
	}
	return "·", theme.Muted
}
