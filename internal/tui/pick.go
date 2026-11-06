package tui

import "slices"

// pickRow is a step of the list that a run is chosen from.
type pickRow struct {
	stepRow
	// chosen is the row's own box.
	chosen bool
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
	return r.chosen
}

// box draws the row's checkbox.
func (r pickRow) box() string {
	if r.chosen {
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
			l.rows = append(l.rows, pickRow{stepRow: stepRow{parent: g.Name, index: i, name: name}})
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
