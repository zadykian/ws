package queue

import (
	"slices"

	"github.com/zadykian/ws/engine/tui/internal/list"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// settled returns the queue with its boxes and locks worked out again. A stage with steps of its own
// is on where one is, a locked stage holds its steps, and one walk from the end reaches a whole chain.
func (q queue) settled() queue {
	q.stages = slices.Clone(q.stages)
	for i := range q.stages {
		s := &q.stages[i]
		s.list = s.list.Settled()
		s.row.LockedBy = nil
		if len(s.list.Rows) > 0 {
			s.row.Chosen = slices.ContainsFunc(s.list.Rows, func(r list.PickRow) bool { return r.Chosen })
		}
	}
	for i := len(q.stages) - 1; i >= 0; i-- {
		r := q.stages[i].row
		if r.WaitsOn == "" || !r.On() {
			continue
		}
		for j := range q.stages {
			other := &q.stages[j].row
			if other.Name == r.WaitsOn && !slices.Contains(other.LockedBy, r.Name) {
				other.LockedBy = append(other.LockedBy, r.Name)
			}
		}
	}
	for i, s := range q.stages {
		if len(s.row.LockedBy) > 0 {
			q.stages[i].list = s.list.Held(s.row.LockedBy)
		}
	}
	return q
}

// box draws the checkbox of stage i. It says "[-]" where the stage runs some of its steps.
func (q queue) box(i int) string {
	s := q.stages[i]
	if on, _, all := s.list.Tally(); on > 0 && on < all {
		return "[-]"
	}
	return s.row.Box()
}

// toggled returns the queue with the box under the cursor of the side turned. A stage's or a role's
// turns its shown steps on, or off where all are on. A locked stage or step holds.
func (q queue) toggled(side string, at widget.Cursor) queue {
	lines := q.lines(side)
	if int(at) >= len(lines) {
		return q
	}
	line := lines[at]
	q.stages = slices.Clone(q.stages)
	s := &q.stages[line.stage]
	switch {
	case line.line.Heading:
		parent := s.list.Rows[line.line.Row].Parent
		s.list = s.list.Turned(func(o list.PickRow) bool { return o.Parent == parent })
	case line.head && len(s.row.LockedBy) > 0:
		return q
	case line.head && len(s.list.Rows) > 0:
		s.list = s.list.Turned(func(list.PickRow) bool { return true })
	case line.head:
		s.row.Chosen = !s.row.Chosen
	default:
		row := s.list.Rows[line.line.Row]
		if len(row.LockedBy) > 0 {
			return q
		}
		s.list = s.list.Chose(func(o list.PickRow) bool { return o.SameChoice(row) != o.Chosen })
	}
	return q.settled()
}

// only returns the stages on, as Options.Only names them: a stage runs what is under it.
func (q queue) only() []string {
	var names []string
	for _, s := range q.stages {
		if s.row.On() {
			names = append(names, s.row.Name)
		}
	}
	return names
}

// partial returns what each stage that runs in part runs, by Options.Only's names.
func (q queue) partial() map[string][]string {
	var parts map[string][]string
	for _, s := range q.stages {
		if on, _, all := s.list.Tally(); on > 0 && on < all {
			if parts == nil {
				parts = map[string][]string{}
			}
			parts[s.row.Name] = s.list.Entries(func(list.PickRow) bool { return true })
		}
	}
	return parts
}

// ticked returns the stages that run every step by a tick: a lock alone leaves the steps to the last run.
func (q queue) ticked() []string {
	var names []string
	for _, s := range q.stages {
		on, _, all := s.list.Tally()
		if on == all && slices.ContainsFunc(s.list.Rows, func(r list.PickRow) bool { return r.Chosen }) {
			names = append(names, s.row.Name)
		}
	}
	return names
}

// count returns how many stages are on, how many the scan has reached, and how many there are.
func (q queue) count() (on, scanned, all int) {
	for _, s := range q.stages {
		all++
		if s.row.On() {
			on++
		}
		if s.row.State != list.StepOpen {
			scanned++
		}
	}
	return on, scanned, all
}

// dueAt says that a step in state has work. In a clean stage, a step no scan named has none.
func dueAt(state list.StepState, clean bool) bool {
	switch state {
	case list.StepSame, list.StepLeftOut:
		return false
	case list.StepOpen:
		return !clean
	}
	return true
}

// opened returns the queue as a scan's end leaves it. What is due starts chosen, as the wizard left
// alone runs each stage it reaches, and a check stops short on a new Mac.
func (q queue) opened() queue {
	due := func(s queueStage) bool { return dueAt(s.row.State, false) }
	q.stages = slices.Clone(q.stages)
	for i, s := range q.stages {
		q.stages[i].row.Chosen = due(s)
		q.stages[i].list.Rows = slices.Clone(s.list.Rows)
		for j, r := range s.list.Rows {
			q.stages[i].list.Rows[j].Chosen = dueAt(r.State, s.row.State == list.StepSame)
		}
	}
	return q.settled().whole(due)
}

// whole turns every step on in a stage that want takes and has none on: ws's run can be due, its roles clean.
func (q queue) whole(want func(queueStage) bool) queue {
	q.stages = slices.Clone(q.stages)
	for i, s := range q.stages {
		if on, _, _ := s.list.Tally(); on == 0 && want(s) {
			q.stages[i].list = s.list.Boxed(true)
		}
	}
	return q.settled()
}

// passedOver returns the roles of stage i with changes left out ahead of a step on, within its stage.
func (q queue) passedOver(i int) []string {
	return q.stages[i].list.PassedOver()
}

// passed returns the roles with changes left out ahead of a step on, within their stages.
func (q queue) passed() []string {
	var roles []string
	for i := range q.stages {
		for _, role := range q.passedOver(i) {
			if !slices.Contains(roles, role) {
				roles = append(roles, role)
			}
		}
	}
	return roles
}

// chose returns the queue with the box of each shown row set by box, the stages' own among them. What
// a filter hides keeps its box.
func (q queue) chose(box func(list.PickRow) bool) queue {
	q.stages = slices.Clone(q.stages)
	for i, s := range q.stages {
		if s.list.Named(s.row.Name) {
			q.stages[i].row.Chosen = box(s.row)
		}
		q.stages[i].list = s.list.Chose(box)
	}
	return q.settled()
}
