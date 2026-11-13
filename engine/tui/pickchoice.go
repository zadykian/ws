package tui

import "slices"

// chose returns the list with the box of each shown step set by box. The steps that a filter hides
// keep theirs, as a bulk choice holds what the maintainer sees.
func (l pickList) chose(box func(pickRow) bool) pickList {
	l.rows = slices.Clone(l.rows)
	for i, r := range l.rows {
		if l.shows(r) {
			l.rows[i].chosen = box(r)
		}
	}
	return l.settled()
}

// toggled returns the list with the box under the cursor turned. A role's turns its shown steps on,
// or off where all are on, and so does a stage's. A locked step holds, as the step that waits on it
// would go with it.
func (l pickList) toggled(at Cursor) pickList {
	line, ok := l.current(at)
	if !ok {
		return l
	}
	row := l.rows[line.row]
	switch {
	case line.heading:
		return l.turned(func(o pickRow) bool { return o.parent == row.parent })
	case len(row.lockedBy) > 0:
		return l
	}
	return l.chose(func(o pickRow) bool {
		if o.sameChoice(row) {
			return !row.chosen
		}
		return o.chosen
	})
}

// turned returns the list with the shown steps of part on, or off where all are on.
func (l pickList) turned(part func(pickRow) bool) pickList {
	on := slices.ContainsFunc(l.rows, func(o pickRow) bool { return part(o) && l.shows(o) && !o.on() })
	return l.chose(func(o pickRow) bool {
		if part(o) {
			return on
		}
		return o.chosen
	})
}

// count returns how many of the steps in are on, and how many steps it takes.
func (l pickList) count(in func(pickRow) bool) (on, all int) {
	for _, r := range l.rows {
		if in(r) {
			all++
			if r.on() {
				on++
			}
		}
	}
	return on, all
}

// tally counts the steps on, the steps the scan has reached and the steps.
func (l pickList) tally() (on, scanned, all int) {
	on, all = l.count(func(pickRow) bool { return true })
	_, scanned = l.count(func(r pickRow) bool { return r.state != stepOpen })
	return on, scanned, all
}
