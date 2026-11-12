package tui

import (
	"fmt"
	"slices"
	"strings"
)

// queueStage is a stage of a queue: its own row, the side that shows it, and a list of what it runs.
type queueStage struct {
	row  pickRow
	side Side
	list pickList
}

// queue is the stages in the order they run, each with a list of its own.
type queue struct {
	stages []queueStage
	// heard holds what the scan printed since a step last ended.
	heard []Line
	// words is the filter the lists show under.
	words string
}

// queueOf returns the queue of stages. A group named as its stage has no heading, and one with no
// steps stands for a role no scan has read.
func queueOf(stages []Stage) queue {
	var q queue
	for i, s := range stages {
		groups := make([]listGroup, len(s.Nested))
		for j, g := range s.Nested {
			groups[j] = listGroup{Group: g}
			switch {
			case g.Name == s.Name:
				groups[j].form = groupBare
			case len(g.Steps) == 0:
				groups[j].form = groupUnread
			}
		}
		row := pickRow{stepRow: stepRow{index: i, name: s.Name}, waitsOn: s.Waits}
		q.stages = append(q.stages, queueStage{row: row, side: s.Side, list: newFormedList(groups)})
	}
	return q
}

// queueLine is a line of a queue: a stage's own, or one of its list's.
type queueLine struct {
	stage int
	head  bool
	line  pickLine
}

// lines returns what the filter leaves of the side. A stage whose name the words miss loses its own
// line, and keeps the steps they match.
func (q queue) lines(side Side) []queueLine {
	var lines []queueLine
	for i, s := range q.stages {
		if s.side != side {
			continue
		}
		if s.list.named(s.row.name) {
			lines = append(lines, queueLine{stage: i, head: true})
		}
		for _, line := range s.list.lines() {
			lines = append(lines, queueLine{stage: i, line: line})
		}
	}
	return lines
}

// render draws a line: a stage by its number, and its steps past the room of that number.
func (q queue) render(theme Theme, l queueLine) string {
	s := q.stages[l.stage]
	if !l.head {
		return strings.Repeat(" ", stageColumns) + s.list.render(theme, l.line)
	}
	mark, style := s.row.state.look(theme)
	return fmt.Sprintf("%*d ", stageColumns-3, l.stage+1) + q.box(l.stage) + " " + style.Render(mark) + " " + s.row.name
}

// filtered returns the queue under words.
func (q queue) filtered(words string) queue {
	q.words = words
	q.stages = slices.Clone(q.stages)
	for i := range q.stages {
		q.stages[i].list = q.stages[i].list.filtered(words)
	}
	return q
}

// narrowed returns the queue under words, and the cursor on its line of the side while that shows.
func (q queue) narrowed(side Side, at cursor, words string) (queue, cursor) {
	was := q.lines(side)
	q = q.filtered(words)
	return q, kept(at, was, q.lines(side))
}

// view draws height lines of the side, the cursor's gutter where shown.
func (q queue) view(theme Theme, side Side, height int, at cursor, shown bool) string {
	lines := q.lines(side)
	return windowed(theme, height, at, shown, len(lines), func(i int) string { return q.render(theme, lines[i]) })
}
