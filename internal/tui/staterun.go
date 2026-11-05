package tui

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"
)

// errStateLeft ends a state screen's ctx for a person who left it, which is no failure.
var errStateLeft = errors.New("the state screen was left")

// StateReader reads a machine and changes nothing. b gives the read's effects, for its lines to
// reach the screen and not the terminal under it.
type StateReader func(ctx context.Context, b *Bridge) ([]StateSection, error)

// RunState draws the state screen on c.Out until q, and reads the machine in Run's work, so that a
// read's panic and a signal end as a run's do. A read goes on under the screen, which takes the keys
// meanwhile, and q ends it through ctx. It returns nil after q, and Cut's error after a cut.
func RunState(ctx context.Context, c Config, title string, read StateReader) error {
	ctx, leave := context.WithCancelCause(ctx)
	defer leave(nil)
	// The screen asks once a read has ended, so that one place is enough for an ask never to wait.
	asked := make(chan struct{}, 1)
	ask := func() tea.Msg { asked <- struct{}{}; return nil }
	left := func() tea.Msg { leave(errStateLeft); return nil }
	screen := NewStateScreen(c.Theme, title, ask, left)
	return Run(ctx, c, screen, func(ctx context.Context, b *Bridge) error {
		for {
			select {
			case <-ctx.Done():
				if errors.Is(context.Cause(ctx), errStateLeft) {
					return nil
				}
				return ctx.Err()
			case <-asked:
				sections, err := read(ctx, b)
				if ctx.Err() == nil {
					b.send(StateRead{sections, err})
				}
			}
		}
	})
}
