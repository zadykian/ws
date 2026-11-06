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
// or off where all are on. A locked step holds, as the step that waits on it would go with it.
func (l pickList) toggled() pickList {
	line, ok := l.current()
	if !ok {
		return l
	}
	row := l.rows[line.row]
	if line.heading {
		off := func(o pickRow) bool { return o.parent == row.parent && l.shows(o) && !o.on() }
		turned := slices.ContainsFunc(l.rows, off)
		return l.chose(func(o pickRow) bool {
			if o.parent == row.parent {
				return turned
			}
			return o.chosen
		})
	}
	if len(row.lockedBy) > 0 {
		return l
	}
	return l.chose(func(o pickRow) bool {
		if o.sameChoice(row) {
			return !row.chosen
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
