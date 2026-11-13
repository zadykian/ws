package tui

import "context"

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
	Side string
	// left is the screen as the maintainer left it, for Reopen and NewQueueRun.
	left queueScreen
}

// PickStages draws a queue's lists over scan, as Pick draws a list. scan's Bridge.Report fills
// a stage from what its machine says. A QueueChoice that runs nothing, or goes to the form or the state screen, says it by its Next.
func PickStages(ctx context.Context, c Config, title string, stages []Stage, scan Work) (QueueChoice, error) {
	picks := make(chan QueueChoice, 1)
	return pick(ctx, c, openQueue(title, c.Theme, stages, picks), picks, scan)
}

// Reopen draws the lists again as left, a NextState choice, found them, and returns the next choice. No
// scan runs, as left came past a scan's end and the state screen changes nothing. What the scan
// found stands, and so do the boxes, which a scan would choose again.
func Reopen(ctx context.Context, c Config, left QueueChoice) (QueueChoice, error) {
	picks := make(chan QueueChoice, 1)
	screen := left.left
	screen.picks = picks
	return pick(ctx, c, screen, picks, func(ctx context.Context, _ *Bridge) error {
		<-ctx.Done()
		return nil
	})
}
