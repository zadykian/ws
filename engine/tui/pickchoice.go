package tui

import (
	"slices"

	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// Chose returns the list with the box of each shown step set by box. The steps that a filter hides
// keep theirs, as a bulk choice holds what the maintainer sees.
func (l PickList) Chose(box func(PickRow) bool) PickList {
	l.Rows = slices.Clone(l.Rows)
	for i, r := range l.Rows {
		if l.shows(r) {
			l.Rows[i].Chosen = box(r)
		}
	}
	return l.Settled()
}

// Toggled returns the list with the box under the cursor turned. A role's turns its shown steps on,
// or off where all are on, and so does a stage's. A locked step holds, as the step that waits on it
// would go with it.
func (l PickList) Toggled(at widget.Cursor) PickList {
	line, ok := l.Current(at)
	if !ok {
		return l
	}
	row := l.Rows[line.Row]
	switch {
	case line.Heading:
		return l.Turned(func(o PickRow) bool { return o.Parent == row.Parent })
	case len(row.LockedBy) > 0:
		return l
	}
	return l.Chose(func(o PickRow) bool {
		if o.SameChoice(row) {
			return !row.Chosen
		}
		return o.Chosen
	})
}

// Turned returns the list with the shown steps of part on, or off where all are on.
func (l PickList) Turned(part func(PickRow) bool) PickList {
	on := slices.ContainsFunc(l.Rows, func(o PickRow) bool { return part(o) && l.shows(o) && !o.On() })
	return l.Chose(func(o PickRow) bool {
		if part(o) {
			return on
		}
		return o.Chosen
	})
}

// count returns how many of the steps in are on, and how many steps it takes.
func (l PickList) count(in func(PickRow) bool) (on, all int) {
	for _, r := range l.Rows {
		if in(r) {
			all++
			if r.On() {
				on++
			}
		}
	}
	return on, all
}

// Tally counts the steps on, the steps the scan has reached and the steps.
func (l PickList) Tally() (on, scanned, all int) {
	on, all = l.count(func(PickRow) bool { return true })
	_, scanned = l.count(func(r PickRow) bool { return r.State != StepOpen })
	return on, scanned, all
}
