package tui

// Side is the list of a queue that shows a stage: the machine the stage acts on.
type Side int

// The sides of a queue.
const (
	MacSide Side = iota
	WsSide
	sideCount
)

// unreadIndex is a role's in place of a step's, as no event of a run names a role.
const unreadIndex = -1

// Stage is a step of a queue: the two lists show one order, as stages wait on each other across the
// machines and share a connection, so that no two run at once.
type Stage struct {
	Name string
	// Waits names the stage that has to run first.
	Waits string
	Side  Side
	// Nested is what the stage runs, each group by the Parent of its events. A group with no steps
	// stands for what no scan has named yet: a role shows by its name, and a group called as its
	// stage shows nothing until its events do.
	Nested []Group
}

// newQueue returns the list of stages, numbered across both sides in the order they run.
func newQueue(stages []Stage) pickList {
	var l pickList
	for i, s := range stages {
		row := stepRow{index: i, name: s.Name}
		l.rows = append(l.rows, pickRow{stepRow: row, side: s.Side, number: i + 1, waitsOn: s.Waits})
		for _, g := range s.Nested {
			l.rows = append(l.rows, s.rows(g)...)
		}
	}
	return l
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
