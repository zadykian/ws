package tui

import (
	"slices"

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

// stepRow is a step of the list. parent and index are what its events name it by, as a name can
// come twice in a run.
type stepRow struct {
	parent string
	index  int
	name   string
	state  stepState
}

// label names the row in a line of its own.
func (r stepRow) label() string {
	if r.parent == "" {
		return r.name
	}
	return r.parent + ": " + r.name
}

// stepList is a run's steps in the order they run, so that the first one open is the current one.
type stepList []stepRow

// newStepList returns the list of groups, each step open.
func newStepList(groups []Group) stepList {
	var list stepList
	for _, g := range groups {
		for i, name := range g.Steps {
			list = append(list, stepRow{parent: g.Name, index: i, name: name})
		}
	}
	return list
}

// take returns the list with what e says of its step. The list is copied, as a screen's older value
// shares it. A step outside the list moves nothing, and its lines show all the same.
func (l stepList) take(e step.Event) stepList {
	place := slices.IndexFunc(l, func(r stepRow) bool { return r.parent == e.Parent && r.index == e.Index })
	if place < 0 {
		return l
	}
	l = slices.Clone(l)
	l[place].state = stateAfter(e)
	return l
}

// ended returns how many steps have ended.
func (l stepList) ended() int {
	done := 0
	for _, r := range l {
		if r.state != stepOpen {
			done++
		}
	}
	return done
}
