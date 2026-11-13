package tui

// stageColumns is the columns of a stage's number, which a row under the stage stands past.
const stageColumns = 5

// Stage is a step of a queue. The lists show one order, as stages wait on each other across the
// machines and share a connection, so that no two run at once.
type Stage struct {
	Name string
	// Waits names the stage that has to run first.
	Waits string
	// Side names the stage's machine, as the title of the pane that lists it.
	Side string
	// Nested is what the stage runs, each group by the Parent of its events. A group with no steps
	// stands for what no scan has named yet. A role shows by its name, and a group called as its
	// stage shows nothing until its events do.
	Nested []Group
}
