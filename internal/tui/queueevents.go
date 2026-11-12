package tui

import (
	"slices"

	"github.com/zadykian/ws/step"
)

// hear keeps a line of the scan for the step that ends next.
func (q queue) hear(line Line) queue {
	q.heard = append(slices.Clip(q.heard), line)
	return q
}

// stageOf returns the stage that e's step belongs to, or -1. A stage's own step names its place, and
// any other step the group that a stage's list holds. A group named as its stage may have no step
// yet. The events of the Prepare step belong to no stage.
func (q queue) stageOf(e step.Event) int {
	if e.Parent == "" {
		if e.Index < 0 || e.Index >= len(q.stages) {
			return -1
		}
		return e.Index
	}
	if i := slices.IndexFunc(q.stages, func(s queueStage) bool { return s.list.holds(e.Parent) }); i >= 0 {
		return i
	}
	return slices.IndexFunc(q.stages, func(s queueStage) bool { return s.row.name == e.Parent })
}

// take returns the queue with how the scan left e's step: chosen where it holds changes. Lines of a
// step outside every stage go with it, as the queue has no place for them.
func (q queue) take(e step.Event) queue {
	if e.Kind == step.Planned || e.Kind == step.Started {
		return q
	}
	said := q.heard
	q.heard = nil
	i := q.stageOf(e)
	if i < 0 {
		return q
	}
	q.stages = slices.Clone(q.stages)
	s := &q.stages[i]
	if e.Parent == "" {
		s.row = s.row.took(e, said)
		return q
	}
	s.list = s.list.joined(e)
	s.list.rows = slices.Clone(s.list.rows)
	place := s.list.place(e)
	s.list.rows[place] = s.list.rows[place].took(e, said)
	s.list = s.list.settled()
	return q
}

// reported returns the queue with stage's list made of sections as they say, in place of the rows it
// had. The filter's words stay.
func (q queue) reported(stage string, sections []StateSection) queue {
	i := slices.IndexFunc(q.stages, func(s queueStage) bool { return s.row.name == stage })
	if i < 0 {
		return q
	}
	var rows []pickRow
	for _, section := range sections {
		form := groupHeaded
		if section.Name == stage {
			form = groupBare
		}
		for j, report := range section.Reports {
			row := reportedRow(section.Name, j, report)
			row.form = form
			rows = append(rows, row)
		}
	}
	q.stages = slices.Clone(q.stages)
	q.stages[i].list.rows = rows
	q.stages[i].list = q.stages[i].list.settled()
	return q
}

// follow returns the queue with how a run left e's step, the side that shows it, and the line for
// the cursor. A step that starts takes it, so that its list shows it, and no other event does: -1,
// and a side that means nothing. The boxes stand, as a run takes no choice.
func (q queue) follow(e step.Event) (queue, Side, int) {
	i := q.stageOf(e)
	if i < 0 {
		return q, MacSide, -1
	}
	q.stages = slices.Clone(q.stages)
	s := &q.stages[i]
	at := queueLine{stage: i, head: true}
	if e.Parent == "" {
		s.row.state, s.row.after = stateAfter(e), e.After
	} else {
		s.list = s.list.joined(e)
		s.list.rows = slices.Clone(s.list.rows)
		place := s.list.place(e)
		s.list.rows[place].state, s.list.rows[place].after = stateAfter(e), e.After
		at = queueLine{stage: i, line: pickLine{row: place}}
	}
	if e.Kind != step.Planned && e.Kind != step.Started {
		return q, s.side, -1
	}
	return q, s.side, max(slices.Index(q.lines(s.side), at), 0)
}
