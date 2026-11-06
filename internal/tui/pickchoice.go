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

// toggled returns the list with the box under the cursor turned. A locked step holds, as the step
// that waits on it would go with it.
func (l pickList) toggled() pickList {
	line, ok := l.current()
	if !ok || line.heading {
		return l
	}
	row := l.rows[line.row]
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
