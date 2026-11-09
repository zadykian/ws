package tui

import (
	"fmt"
	"strings"
)

// Side is the list of a queue that shows a stage: the machine the stage acts on.
type Side int

// The sides of a queue.
const (
	MacSide Side = iota
	WsSide
)

// String names the side as its list's pane does.
func (s Side) String() string {
	if s == MacSide {
		return "Mac"
	}
	return "ws"
}

const (
	// unreadIndex is a role's in place of a step's, as no event of a run names a role.
	unreadIndex = -1
	// stageColumns is the columns of a stage's number, which a row under the stage stands past.
	stageColumns = 5
	// roleColumns is what a role's heading adds to a step of the role.
	roleColumns = 2
)

// Stage is a step of a queue. The two lists show one order, as stages wait on each other across the
// machines and share a connection, so that no two run at once.
type Stage struct {
	Name string
	// Waits names the stage that has to run first.
	Waits string
	Side  Side
	// Nested is what the stage runs, each group by the Parent of its events. A group with no steps
	// stands for what no scan has named yet. A role shows by its name, and a group called as its
	// stage shows nothing until its events do.
	Nested []Group
}

// newQueue returns the list of stages, numbered across the sides in the order they run, and turned
// to the first list.
func newQueue(stages []Stage) pickList {
	var l pickList
	for i, s := range stages {
		row := stepRow{index: i, name: s.Name}
		l.rows = append(l.rows, pickRow{stepRow: row, side: s.Side, number: i + 1, waitsOn: s.Waits})
		for _, g := range s.Nested {
			l.rows = append(l.rows, s.rows(g)...)
		}
	}
	return l.facing(l.sides()[0])
}

// rows returns the rows of a group of the stage. A role unread has one, by which ws's run names it.
func (s Stage) rows(g Group) []pickRow {
	if len(g.Steps) == 0 && g.Name != s.Name {
		return []pickRow{{stepRow: stepRow{index: unreadIndex, name: g.Name}, side: s.Side, stage: s.Name}}
	}
	rows := make([]pickRow, len(g.Steps))
	for i, name := range g.Steps {
		row := stepRow{parent: g.Name, index: i, name: name}
		rows[i] = pickRow{stepRow: row, side: s.Side, stage: s.Name, waitsOn: g.Waits[name]}
	}
	return rows
}

// indent is what stands ahead of the row's box: a stage's number, or the room of its depth.
func (r pickRow) indent() string {
	switch {
	case r.number > 0:
		return fmt.Sprintf("%*d ", stageColumns-3, r.number)
	case r.stage != "" && r.parent != "" && r.parent != r.stage:
		return strings.Repeat(" ", stageColumns+roleColumns)
	case r.stage != "":
		return strings.Repeat(" ", stageColumns)
	case r.parent != "":
		return strings.Repeat(" ", roleColumns)
	}
	return ""
}

// headingIndent is what stands ahead of the box of the row's role.
func (r pickRow) headingIndent() string {
	if r.stage != "" {
		return strings.Repeat(" ", stageColumns)
	}
	return ""
}

// facing returns the list turned to side, under the cursor that side had.
func (l pickList) facing(side Side) pickList {
	if side != l.side {
		l.side, l.at, l.other = side, l.other, l.at
	}
	l.at = max(min(l.at, len(l.lines())-1), 0)
	return l
}

// glance returns the list to draw side by, which shows no cursor out of focus.
func (l pickList) glance(side Side) pickList {
	seen := l.facing(side)
	seen.blind = side != l.side
	return seen
}
