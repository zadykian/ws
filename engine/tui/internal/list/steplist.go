package list

import (
	"slices"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// Group is the steps of a role or a stage, by the names that a run's events give them.
type Group struct {
	// Name is the run's Options.Parent. An empty one draws no heading.
	Name  string
	Steps []string
	// Waits gives the step that a step of Steps waits on, by name, where it has one.
	Waits map[string]string
}

// NewGroup names steps as a run of them under parent will, so that a screen knows each one before
// the run starts.
func NewGroup(parent string, steps []step.Step) Group {
	g := Group{Name: parent, Waits: map[string]string{}}
	for _, s := range steps {
		g.Steps = append(g.Steps, s.Name())
		if awaited := s.WaitsOn(); awaited != "" {
			g.Waits[s.Name()] = awaited
		}
	}
	return g
}

// StepState says how a run left a step.
type StepState int

// The states of a step, from the scan's events.
const (
	StepOpen StepState = iota
	StepSame
	StepChanged
	StepLeftOut
	StepFailed
	StepSkipped
)

// StateAfter is the state that e leaves its step in. Planned opens a step that has ended, as the
// wizard runs its Setup's steps more than once.
func StateAfter(e step.Event) StepState {
	switch e.Kind {
	case step.Done:
		switch {
		case e.LeftOut:
			return StepLeftOut
		case len(e.Changes) != 0:
			return StepChanged
		}
		return StepSame
	case step.Failed:
		return StepFailed
	case step.Skipped:
		if e.LeftOut {
			return StepLeftOut
		}
		return StepSkipped
	case step.Planned, step.Started:
	}
	return StepOpen
}

// Look is the mark and the style of a state. The marks differ by shape, as NO_COLOR takes the rest.
func (s StepState) Look(theme widget.Theme) (string, lipgloss.Style) {
	switch s {
	case StepSame:
		return "✓", theme.Good
	case StepChanged:
		return "●", theme.Warn
	case StepLeftOut:
		return "○", theme.Muted
	case StepFailed:
		return "✗", theme.Bad
	case StepSkipped:
		return "–", theme.Muted
	case StepOpen:
	}
	return "·", theme.Muted
}

// StepRow is a step of the list. parent and index are what its events name it by, as a name can
// come twice in a run.
type StepRow struct {
	Parent string
	Index  int
	Name   string
	State  StepState
}

// Label names the row in a line of its own.
func (r StepRow) Label() string {
	if r.Parent == "" {
		return r.Name
	}
	return r.Parent + ": " + r.Name
}

// render draws the row, indented under a heading.
func (r StepRow) render(theme widget.Theme, current bool) string {
	mark, style := r.State.Look(theme)
	if current && r.State == StepOpen {
		mark, style = "▸", theme.PaneTitle
	}
	indent := ""
	if r.Parent != "" {
		indent = "  "
	}
	return indent + style.Render(mark+" "+r.Name)
}

// StepList is a run's steps in the order they run, so that the first one open is the current one.
type StepList []StepRow

// NewStepList returns the list of groups, each step open.
func NewStepList(groups []Group) StepList {
	var list StepList
	for _, g := range groups {
		for i, name := range g.Steps {
			list = append(list, StepRow{Parent: g.Name, Index: i, Name: name})
		}
	}
	return list
}

// Take returns the list with what e says of its step. The list is copied, as a screen's older value
// shares it. A step outside the list moves nothing, and its lines show all the same.
func (l StepList) Take(e step.Event) StepList {
	place := slices.IndexFunc(l, func(r StepRow) bool { return r.Parent == e.Parent && r.Index == e.Index })
	if place < 0 {
		return l
	}
	l = slices.Clone(l)
	l[place].State = StateAfter(e)
	return l
}

// At returns the place of the step that the run stands on, the failed one past a failure, or -1
// where every step has ended.
func (l StepList) At() int {
	return slices.IndexFunc(l, func(r StepRow) bool { return r.State == StepOpen || r.State == StepFailed })
}

// Ended returns how many steps have ended.
func (l StepList) Ended() int {
	done := 0
	for _, r := range l {
		if r.State != StepOpen {
			done++
		}
	}
	return done
}

// View draws height lines of the list, the steps under their headings. A run has more steps than a
// window has lines, so the current one stays in sight, a third of the way down.
func (l StepList) View(theme widget.Theme, height int) string {
	at := l.At()
	lines := make([]string, 0, len(l))
	sight := 0
	for i, row := range l {
		if row.Parent != "" && (i == 0 || row.Parent != l[i-1].Parent) {
			lines = append(lines, theme.PaneTitle.Render(row.Parent))
		}
		if i == at || at < 0 {
			sight = len(lines)
		}
		lines = append(lines, row.render(theme, i == at))
	}
	top := max(min(sight-height/3, len(lines)-height), 0)
	return strings.Join(lines[top:min(top+height, len(lines))], "\n")
}
