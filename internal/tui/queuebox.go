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

// toggled returns the queue with the box under the cursor of the side turned. A stage's turns its
// shown steps on, or off where all are on, and so does a role's. A locked stage or step holds, as the
// stage that waits on it would go with it.
func (q queue) toggled(side Side, at cursor) queue {
	lines := q.lines(side)
	if int(at) >= len(lines) {
		return q
	}
	line := lines[at]
	q.stages = slices.Clone(q.stages)
	s := &q.stages[line.stage]
	switch {
	case line.line.heading:
		parent := s.list.rows[line.line.row].parent
		s.list = s.list.turned(func(o pickRow) bool { return o.parent == parent })
	case line.head && len(s.row.lockedBy) > 0:
		return q
	case line.head && len(s.list.rows) > 0:
		s.list = s.list.turned(func(pickRow) bool { return true })
	case line.head:
		s.row.chosen = !s.row.chosen
	default:
		row := s.list.rows[line.line.row]
		if len(row.lockedBy) > 0 {
			return q
		}
		s.list = s.list.chose(func(o pickRow) bool { return o.sameChoice(row) != o.chosen })
	}
	return q.settled()
}

// only returns the stages on, as Options.Only names them: a stage runs what is under it.
func (q queue) only() []string {
	var names []string
	for _, s := range q.stages {
		if s.row.on() {
			names = append(names, s.row.name)
		}
	}
	return names
}

// partial returns what each stage that runs in part runs, by Options.Only's names.
func (q queue) partial() map[string][]string {
	var parts map[string][]string
	for _, s := range q.stages {
		if on, _, all := s.list.tally(); on > 0 && on < all {
			if parts == nil {
				parts = map[string][]string{}
			}
			parts[s.row.name] = s.list.entries(func(pickRow) bool { return true })
		}
	}
	return parts
}

// ticked returns the stages that run every step, one of them by a tick. A lock alone asks for the
// stage's end, which a good last run gives, and a tick for the steps, as one can drift after it.
func (q queue) ticked() []string {
	var names []string
	for _, s := range q.stages {
		on, _, all := s.list.tally()
		if on == all && slices.ContainsFunc(s.list.rows, func(r pickRow) bool { return r.chosen }) {
			names = append(names, s.row.name)
		}
	}
	return names
}

// count returns how many stages are on, how many the scan has reached, and how many there are.
func (q queue) count() (on, scanned, all int) {
	for _, s := range q.stages {
		all++
		if s.row.on() {
			on++
		}
		if s.row.state != stepOpen {
			scanned++
		}
	}
	return on, scanned, all
}
