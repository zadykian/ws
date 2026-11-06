package tui

import (
	"cmp"
	"context"
	"errors"
)

// Pick draws the list of groups over scan, a run with --check, and returns what the maintainer
// chose: a Choice with no steps where they left. Ctrl-C ends it as Cut says, and no stop is asked.
func Pick(ctx context.Context, c Config, title string, groups []Group, scan Work) (Choice, error) {
	picks := make(chan Choice, 1)
	return pick(ctx, c, newPickScreen(title, c.Theme, groups, picks), picks, scan)
}

// PickStages draws a queue's two lists over scan, as Pick draws a list. scan's Bridge.Report fills
// a stage from what its machine says. A Choice with neither steps nor Configure says that they left.
func PickStages(ctx context.Context, c Config, title string, stages []Stage, scan Work) (Choice, error) {
	picks := make(chan Choice, 1)
	return pick(ctx, c, newQueueScreen(title, c.Theme, stages, picks), picks, scan)
}

// Report fills a stage of the lists with what its machine says: sections, or problem, why it says
// nothing. It goes ahead of the scan's end, which chooses by it.
func (b *Bridge) Report(stage string, sections []StateSection, problem string) {
	b.send(reportedMsg{stage, sections, problem})
}

// pick runs a list's screen over scan until picks gives a choice.
func pick(ctx context.Context, c Config, screen pickScreen, picks <-chan Choice, scan Work) (Choice, error) {
	var choice Choice
	c.Halts, c.quiet = false, true
	err := Run(ctx, c, screen, func(ctx context.Context, b *Bridge) error {
		var err error
		choice, err = awaitPick(ctx, b, picks, scan)
		return err
	})
	if err != nil {
		return Choice{}, err // A work left behind may still write choice.
	}
	return choice, nil
}

// awaitPick runs scan beside the list and returns what the list gives. A choice ends a scan still
// going and waits for it, as a run beside it would contend for the same machine. A scan that fails
// leaves the list open, with what remains of it to choose from. Only a panic ends the wait.
func awaitPick(ctx context.Context, b *Bridge, picks <-chan Choice, scan Work) (Choice, error) {
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
				return Choice{}, err
			}
			waiting = nil
			msg := scannedMsg{}
			if err != nil {
				msg.problem = err.Error()
			}
			b.send(msg)
		case choice := <-picks:
			cancel()
			return choice, panicOf(<-done)
		case <-ctx.Done():
			return Choice{}, cmp.Or(panicOf(<-done), ctx.Err())
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
