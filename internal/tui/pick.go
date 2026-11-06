package tui

import "slices"

// pickRow is a step of the list that a run is chosen from.
type pickRow struct {
	stepRow
	// waitsOn names the step of its group that has to run first.
	waitsOn string
	// chosen is the row's own box. It stands under a lock, and counts again once the lock goes.
	chosen bool
	// lockedBy names the steps on that wait on this one: settled works it out.
	lockedBy []string
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

// settled returns the list with its locks worked out again. A step waits on an earlier one alone,
// so that one walk from the end reaches the whole of a chain.
func (l pickList) settled() pickList {
	l.rows = slices.Clone(l.rows)
	for i := range l.rows {
		l.rows[i].lockedBy = nil
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
