package queue

import (
	"fmt"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/tui/internal/list"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// queueStage is a stage of a queue: its own row, the side that shows it, and a list of what it runs.
type queueStage struct {
	row  list.PickRow
	side string
	list list.PickList
}

// queue is the stages in the order they run, each with a list of its own.
type queue struct {
	stages []queueStage
	// heard holds what the scan printed since a step last ended.
	heard []widget.Line
	// words is the filter the lists show under.
	words string
	// side is the list in focus.
	side string
}

// queueOf returns the queue of stages. A group named as its stage has no heading, and one with no
// steps stands for a role no scan has read.
func queueOf(stages []Stage) queue {
	var q queue
	for i, s := range stages {
		groups := make([]list.RowGroup, len(s.Nested))
		for j, g := range s.Nested {
			groups[j] = list.RowGroup{Group: g}
			switch {
			case g.Name == s.Name:
				groups[j].Form = list.GroupBare
			case len(g.Steps) == 0:
				groups[j].Form = list.GroupUnread
			}
		}
		row := list.PickRow{StepRow: list.StepRow{Index: i, Name: s.Name}, WaitsOn: s.Waits}
		q.stages = append(q.stages, queueStage{row: row, side: s.Side, list: list.NewFormedList(groups)})
	}
	if sides := q.sides(); len(sides) > 0 {
		q.side = sides[0]
	}
	return q
}

// queueLine is a line of a queue: a stage's own, or one of its list's.
type queueLine struct {
	stage int
	head  bool
	line  list.PickLine
}

// lines returns what the filter leaves of the side. A stage whose name the words miss loses its own
// line, and keeps the steps they match.
func (q queue) lines(side string) []queueLine {
	var lines []queueLine
	for i, s := range q.stages {
		if s.side != side {
			continue
		}
		if s.list.Named(s.row.Name) {
			lines = append(lines, queueLine{stage: i, head: true})
		}
		for _, line := range s.list.Lines() {
			lines = append(lines, queueLine{stage: i, line: line})
		}
	}
	return lines
}

// render draws a line: a stage by its number, and its steps past the room of that number.
func (q queue) render(theme widget.Theme, l queueLine) string {
	s := q.stages[l.stage]
	if !l.head {
		return strings.Repeat(" ", stageColumns) + s.list.Render(theme, l.line)
	}
	mark, style := s.row.State.Look(theme)
	return fmt.Sprintf("%*d ", stageColumns-3, l.stage+1) + q.box(l.stage) + " " + style.Render(mark) + " " + s.row.Name
}

// filtered returns the queue under words.
func (q queue) filtered(words string) queue {
	q.words = words
	q.stages = slices.Clone(q.stages)
	for i := range q.stages {
		q.stages[i].list = q.stages[i].list.Filtered(words)
	}
	return q
}

// narrowed returns the queue under words, and the cursor on its line of the side while that shows.
func (q queue) narrowed(side string, at widget.Cursor, words string) (queue, widget.Cursor) {
	was := q.lines(side)
	q = q.filtered(words)
	return q, widget.Kept(at, was, q.lines(side))
}

// view draws height lines of the side, the cursor's gutter where shown.
func (q queue) view(theme widget.Theme, side string, height int, at widget.Cursor, shown bool) string {
	lines := q.lines(side)
	return list.Windowed(theme, height, at, shown, len(lines), func(i int) string { return q.render(theme, lines[i]) })
}
