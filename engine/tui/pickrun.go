package tui

import (
	"cmp"
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/tui/internal/list"
)

// Pick draws the list of groups over scan, a run with --check, and returns what the maintainer
// chose: a Choice with no steps where they left. Ctrl-C ends it as Cut says, and no stop is asked.
func Pick(ctx context.Context, c Config, title string, groups []list.Group, scan Work) (Choice, error) {
	picks := make(chan Choice, 1)
	return pick(ctx, c, newPickScreen(title, c.Theme, groups, picks), picks, scan)
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

// Report fills a stage of the lists with what its machine says: sections, or problem, why it says
// nothing. It goes ahead of the scan's end, which chooses by it.
func (b *Bridge) Report(stage string, sections []StateSection, problem string) {
	b.send(reportedMsg{stage, sections, problem})
}

// pick runs a list's screen over scan until picks gives a choice, of whatever type the screen leaves.
func pick[C any](ctx context.Context, c Config, screen tea.Model, picks <-chan C, scan Work) (C, error) {
	var choice C
	c.quiet = true
	err := Run(ctx, c, screen, func(ctx context.Context, b *Bridge) error {
		var err error
		choice, err = awaitPick(ctx, b, picks, scan)
		return err
	})
	if err != nil {
		var none C
		return none, err // A work left behind may still write choice.
	}
	return choice, nil
}

// awaitPick runs scan beside the list and returns what the list gives. A choice ends a scan still
// going and waits for it, as a run beside it would contend for the same machine. A scan that fails
// leaves the list open, with what remains of it to choose from. Only a panic ends the wait.
func awaitPick[C any](ctx context.Context, b *Bridge, picks <-chan C, scan Work) (C, error) {
	var none C
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		defer close(done)
		done <- guarded(ctx, b, scan)
	}()
	waiting := done
	for {
		select {
		case err := <-waiting:
			if errors.Is(err, errPanic) {
				return none, err
			}
			waiting = nil
			msg := list.ScannedMsg{}
			if err != nil {
				msg.Problem = err.Error()
			}
			b.send(msg)
		case choice := <-picks:
			cancel()
			return choice, panicOf(<-done)
		case <-ctx.Done():
			return none, cmp.Or(panicOf(<-done), ctx.Err())
		}
	}
}

// panicOf keeps a panic, the one end of a scan that a list's end must not hide.
func panicOf(err error) error {
	if errors.Is(err, errPanic) {
		return err
	}
	return nil
}
