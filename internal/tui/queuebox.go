package tui

import "slices"

// settled returns the queue with its boxes and locks worked out again. A stage with steps of its own
// is on where one is. A stage waits on an earlier one alone, so that one walk from the end reaches
// the whole of a chain, and a locked stage holds its steps.
func (q queue) settled() queue {
	q.stages = slices.Clone(q.stages)
	for i := range q.stages {
		s := &q.stages[i]
		s.list = s.list.settled()
		s.row.lockedBy = nil
		if len(s.list.rows) > 0 {
			s.row.chosen = slices.ContainsFunc(s.list.rows, func(r pickRow) bool { return r.chosen })
		}
	}
	for i := len(q.stages) - 1; i >= 0; i-- {
		r := q.stages[i].row
		if r.waitsOn == "" || !r.on() {
			continue
		}
		for j := range q.stages {
			other := &q.stages[j].row
			if other.name == r.waitsOn && !slices.Contains(other.lockedBy, r.name) {
				other.lockedBy = append(other.lockedBy, r.name)
			}
		}
	}
	for i, s := range q.stages {
		if len(s.row.lockedBy) > 0 {
			q.stages[i].list = s.list.held(s.row.lockedBy)
		}
	}
	return q
}

// box draws the checkbox of stage i. It says "[-]" where the stage runs some of its steps.
func (q queue) box(i int) string {
	s := q.stages[i]
	if on, _, all := s.list.tally(); on > 0 && on < all {
		return "[-]"
	}
	return s.row.box()
}
