package tui

import (
	"slices"
	"strings"

	"github.com/zadykian/ws/step"
)

// pickRow is a step of the list that a run is chosen from.
type pickRow struct {
	stepRow
	// waitsOn names the step of its group that has to run first.
	waitsOn string
	// chosen is the row's own box. It stands under a lock, and counts again once the lock goes.
	chosen bool
	// lockedBy names the steps on that wait on this one: settled works it out.
	lockedBy []string
	// after names the step whose changes kept the scan from planning this one.
	after string
	// said holds what the scan printed of the step.
	said []Line
}

// entry names the row as Options.Only does.
func (r pickRow) entry() string {
	if r.parent == "" {
		return r.name
	}
	return r.parent + "/" + r.name
}

// on says that the step runs.
func (r pickRow) on() bool {
	return r.chosen || len(r.lockedBy) > 0
}

// pending says that the step has changes to apply, or waits on a step that has.
func (r pickRow) pending() bool {
	return r.state == stepChanged || r.after != ""
}

// sameChoice says that o is r to a run, which chooses by name, as a name can come twice in one.
func (r pickRow) sameChoice(o pickRow) bool {
	return r.parent == o.parent && r.name == o.name
}

// box draws the row's checkbox. A locked step takes no key, and its box says so by shape.
func (r pickRow) box() string {
	switch {
	case len(r.lockedBy) > 0:
		return "[#]"
	case r.chosen:
		return "[x]"
	}
	return "[ ]"
}

// pickList is the steps of a run to choose from, in the order they run. Rows are copied on write, as
// a screen's older value shares them.
type pickList struct {
	rows []pickRow
	// heard holds what the scan printed since a step last ended.
	heard []Line
	// at is the cursor's place in lines.
	at int
}

// newPickList returns the list of groups, each step open and left out.
func newPickList(groups []Group) pickList {
	var l pickList
	for _, g := range groups {
		for i, name := range g.Steps {
			row := stepRow{parent: g.Name, index: i, name: name}
			l.rows = append(l.rows, pickRow{stepRow: row, waitsOn: g.Waits[name]})
		}
	}
	return l
}

// hear keeps a line of the scan for the step that ends next.
func (l pickList) hear(line Line) pickList {
	l.heard = append(slices.Clip(l.heard), line)
	return l
}

// take returns the list with how the scan left e's step: chosen where it holds changes. Lines of a
// step outside the list go with it, as the list has no place for them.
func (l pickList) take(e step.Event) pickList {
	if e.Kind == step.Planned || e.Kind == step.Started {
		return l
	}
	said := l.heard
	l.heard = nil
	place := slices.IndexFunc(l.rows, func(r pickRow) bool { return r.parent == e.Parent && r.index == e.Index })
	if place < 0 {
		return l
	}
	l.rows = slices.Clone(l.rows)
	row := &l.rows[place]
	row.state, row.after = stateAfter(e), e.After
	row.chosen = row.pending()
	row.said = nil
	if e.Kind != step.Skipped {
		row.said = withoutLabel(said, row.label())
	}
	if e.Err != nil {
		row.said = append(row.said, Line{Warned, e.Err.Error()})
	}
	return l.settled()
}

// withoutLabel returns lines less the step's name that opens them, as the pane they go to has it in
// its title.
func withoutLabel(lines []Line, label string) []Line {
	kept := make([]Line, len(lines))
	for i, line := range lines {
		kept[i] = Line{line.Kind, strings.TrimPrefix(line.Text, label+": ")}
	}
	return kept
}

// settled returns the list with its locks worked out again, and a box ticked on every step that
// its name runs. A step waits on an earlier one alone, so that one walk from the end reaches the
// whole of a chain.
func (l pickList) settled() pickList {
	l.rows = slices.Clone(l.rows)
	for i, r := range l.rows {
		l.rows[i].lockedBy = nil
		for j := range l.rows {
			l.rows[j].chosen = l.rows[j].chosen || r.chosen && r.sameChoice(l.rows[j])
		}
	}
	for i := len(l.rows) - 1; i >= 0; i-- {
		r := l.rows[i]
		if r.waitsOn == "" || !r.on() {
			continue
		}
		for j := range l.rows {
			other := &l.rows[j]
			if other.parent == r.parent && other.name == r.waitsOn && !slices.Contains(other.lockedBy, r.name) {
				other.lockedBy = append(other.lockedBy, r.name)
			}
		}
	}
	return l
}

// only returns the steps on, as Options.Only names them. A name that comes twice in a run is one
// choice there, so it comes once here.
func (l pickList) only() []string {
	var entries []string
	for _, r := range l.rows {
		if r.on() && !slices.Contains(entries, r.entry()) {
			entries = append(entries, r.entry())
		}
	}
	return entries
}

// passedOver returns the roles that hold changes left out ahead of a step on. The order of roles
// has no model, as no step names a role it needs: a later one may fail without them, or may not,
// so the list warns and locks nothing.
func (l pickList) passedOver() []string {
	var held, named []string
	for _, r := range l.rows {
		if r.parent != "" && r.pending() && !r.on() && !slices.Contains(held, r.parent) {
			held = append(held, r.parent)
		}
		if !r.on() {
			continue
		}
		for _, parent := range held {
			if parent != r.parent && !slices.Contains(named, parent) {
				named = append(named, parent)
			}
		}
	}
	return named
}
