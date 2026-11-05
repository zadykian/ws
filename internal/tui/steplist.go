package tui

import (
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
