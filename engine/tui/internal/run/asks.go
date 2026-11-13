package run

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"
)

// errAsksLeft ends a screen's ctx for a person who left it, which is no failure.
var errAsksLeft = errors.New("the screen was left")

// Asks serves a screen's asks one at a time, off the screen's goroutine, and sends each reply back
// as a message. What an ask needs, such as a form's edits, stays in its closure: they can hold a secret.
type Asks struct {
	queue chan func(context.Context, *Bridge) tea.Msg
	done  chan struct{}
	end   context.CancelCauseFunc
}

// NewAsks returns asks whose leave calls end, the cancel of Serve's ctx.
func NewAsks(end context.CancelCauseFunc) *Asks {
	return &Asks{queue: make(chan func(context.Context, *Bridge) tea.Msg), done: make(chan struct{}), end: end}
}

// Ask hands f to Serve and returns once Serve took it, or at once past Serve's end, so that the
// screen's command never hangs.
func (a *Asks) Ask(f func(context.Context, *Bridge) tea.Msg) {
	select {
	case a.queue <- f:
	case <-a.done:
	}
}

// Leave ends ctx with errAsksLeft, and as a tea.Cmd returns nil.
func (a *Asks) Leave() tea.Msg {
	a.end(errAsksLeft)
	return nil
}

// Serve runs each ask in turn and sends its reply while ctx lives. It returns nil after Leave, and
// ctx's error after any other end, which is the error a cut run ends with. A reply past ctx's end is
// dropped, as its screen has gone.
func (a *Asks) Serve(ctx context.Context, b *Bridge) error {
	defer close(a.done)
	for {
		select {
		case <-ctx.Done():
			if errors.Is(context.Cause(ctx), errAsksLeft) {
				return nil
			}
			return ctx.Err()
		case f := <-a.queue:
			if msg := f(ctx, b); ctx.Err() == nil {
				b.send(msg)
			}
		}
	}
}
