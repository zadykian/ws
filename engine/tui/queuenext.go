package tui

// Next is what the queue's screen was left for.
type Next int

// The ways to leave the queue's screen. NextLeave is the zero value: Esc, q or Ctrl-C.
const (
	NextLeave Next = iota
	// NextRun runs the stages that are chosen.
	NextRun
	// NextCheck plans them and applies nothing.
	NextCheck
	// NextConfigure leaves for the values' form, whatever else is chosen.
	NextConfigure
	// NextState leaves for the state screen of Side's machine, and Reopen comes back to it.
	NextState
)

// QueueChoice is what the queue's screen leaves to the wizard's loop.
type QueueChoice struct {
	Next Next
	// Only names the stages to run, in order, and the steps of a stage that run in part, by entry.
	Only []string
	// Partial gives, by stage, the steps on where some of the stage's are off, by entry. A stage it
	// has no key for runs whole.
	Partial map[string][]string
	// Whole names the stages that run every step, one of them by a tick. Such a stage is to run
	// though its last run passed. One that a lock alone holds whole isn't.
	Whole []string
	// Side is the pane that faced the maintainer.
	Side Side
	// left is the screen as the maintainer left it, for Reopen and NewQueueRun.
	left queueScreen
}
