package tui

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
