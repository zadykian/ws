package tui

import (
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// PickRow is a step of the list that a run is chosen from.
type PickRow struct {
	StepRow
	// Form is how the row's group shows: groupHeaded by the zero value.
	Form groupForm
	// WaitsOn names the step of its group that has to run first.
	WaitsOn string
	// Chosen is the row's own box. It stands under a lock, and counts again once the lock goes.
	Chosen bool
	// LockedBy names the steps on that wait on this one: settled works it out.
	LockedBy []string
	// After names the step whose changes kept the scan from planning this one.
	After string
	// Said holds what the scan printed of the step.
	Said []widget.Line
}

// entry names the row as Options.Only does.
func (r PickRow) entry() string {
	if r.Parent == "" {
		return r.Name
	}
	return r.Parent + "/" + r.Name
}

// On says that the step runs.
func (r PickRow) On() bool {
	return r.Chosen || len(r.LockedBy) > 0
}

// Pending says that the step has changes to apply, or waits on a step that has.
func (r PickRow) Pending() bool {
	return r.State == StepChanged || r.After != ""
}

// SameChoice says that o is r to a run, which chooses by name, as a name can come twice in one.
func (r PickRow) SameChoice(o PickRow) bool {
	return r.Parent == o.Parent && r.Name == o.Name
}

// Box draws the row's checkbox. A locked step takes no key, and its box says so by shape.
func (r PickRow) Box() string {
	switch {
	case len(r.LockedBy) > 0:
		return "[#]"
	case r.Chosen:
		return "[x]"
	}
	return "[ ]"
}

// PickList is the steps of a run to choose from, in the order they run. Rows are copied on write, as
// a screen's older value shares them.
type PickList struct {
	Rows []PickRow
	// heard holds what the scan printed since a step last ended.
	heard []widget.Line
	// Words is the filter: the list shows the steps whose entry holds them.
	Words string
}

// NewPickList returns the list of groups, each step open and left out, each under its heading.
func NewPickList(groups []Group) PickList {
	formed := make([]ListGroup, len(groups))
	for i, g := range groups {
		formed[i] = ListGroup{Group: g}
	}
	return NewFormedList(formed)
}

// Hear keeps a line of the scan for the step that ends next.
func (l PickList) Hear(line widget.Line) PickList {
	l.heard = append(slices.Clip(l.heard), line)
	return l
}

// Take returns the list with how the scan left e's step: chosen where it holds changes. Lines of a
// step outside the list go with it, as the list has no place for them.
func (l PickList) Take(e step.Event) PickList {
	if e.Kind == step.Planned || e.Kind == step.Started {
		return l
	}
	said := l.heard
	l.heard = nil
	place := l.Place(e)
	if place < 0 {
		return l
	}
	l.Rows = slices.Clone(l.Rows)
	l.Rows[place] = l.Rows[place].Took(e, said)
	return l.Settled()
}

// Took returns the row as e leaves its step: chosen where it holds changes. said is what the scan
// printed since a step last ended.
func (r PickRow) Took(e step.Event, said []widget.Line) PickRow {
	r.State, r.After = StateAfter(e), e.After
	r.Chosen = r.Pending()
	r.Said = nil
	if e.Kind != step.Skipped {
		r.Said = withoutLabel(said, r.Label())
	}
	if e.Err != nil {
		r.Said = append(r.Said, widget.Line{Kind: widget.Warned, Text: e.Err.Error()})
	}
	return r
}

// Place returns where the row of e's step stands, or -1.
func (l PickList) Place(e step.Event) int {
	return slices.IndexFunc(l.Rows, func(r PickRow) bool { return r.Parent == e.Parent && r.Index == e.Index })
}

// withoutLabel returns lines less the step's name that opens them, as the pane they go to has it in
// its title.
func withoutLabel(lines []widget.Line, label string) []widget.Line {
	kept := make([]widget.Line, len(lines))
	for i, line := range lines {
		kept[i] = widget.Line{Kind: line.Kind, Text: strings.TrimPrefix(line.Text, label+": ")}
	}
	return kept
}

// Settled returns the list with its locks worked out again, and a box ticked on every step that
// its name runs. A step waits on an earlier one alone, so that one walk from the end reaches the
// whole of a chain.
func (l PickList) Settled() PickList {
	l.Rows = slices.Clone(l.Rows)
	for i, r := range l.Rows {
		l.Rows[i].LockedBy = nil
		for j := range l.Rows {
			l.Rows[j].Chosen = l.Rows[j].Chosen || r.Chosen && r.SameChoice(l.Rows[j])
		}
	}
	for i := len(l.Rows) - 1; i >= 0; i-- {
		r := l.Rows[i]
		if r.WaitsOn == "" || !r.On() {
			continue
		}
		for j := range l.Rows {
			other := &l.Rows[j]
			if other.Parent == r.Parent && other.Name == r.WaitsOn && !slices.Contains(other.LockedBy, r.Name) {
				other.LockedBy = append(other.LockedBy, r.Name)
			}
		}
	}
	return l
}

// Only returns the steps on, as Options.Only names them. A name that comes twice in a run is one
// choice there, so it comes once here.
func (l PickList) Only() []string {
	return l.Entries(func(PickRow) bool { return true })
}

// Entries returns those rows in on, each name once.
func (l PickList) Entries(in func(PickRow) bool) []string {
	var entries []string
	for _, r := range l.Rows {
		if in(r) && r.On() && !slices.Contains(entries, r.entry()) {
			entries = append(entries, r.entry())
		}
	}
	return entries
}

// PassedOver returns the roles that hold changes left out ahead of a step on. The order of roles
// has no model, as no step names a role it needs. A later one may fail without them, or may not, so
// the list warns and locks nothing.
func (l PickList) PassedOver() []string {
	var held, named []string
	for _, r := range l.Rows {
		if r.Parent != "" && r.Pending() && !r.On() && !slices.Contains(held, r.Parent) {
			held = append(held, r.Parent)
		}
		if !r.On() {
			continue
		}
		for _, parent := range held {
			if parent != r.Parent && !slices.Contains(named, parent) {
				named = append(named, parent)
			}
		}
	}
	return named
}
