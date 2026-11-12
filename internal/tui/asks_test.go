package tui

import (
	"context"
	"io"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// servedAsks starts serve, with replies on a channel and its result on another.
func servedAsks(t *testing.T) (*asks, context.CancelCauseFunc, chan tea.Msg, chan error) {
	ctx, end := context.WithCancelCause(t.Context())
	a := newAsks(end)
	b, _ := testBridge(io.Discard)
	replies := make(chan tea.Msg, 8)
	b.send = func(msg tea.Msg) { replies <- msg }
	ended := make(chan error, 1)
	go func() { ended <- a.serve(ctx, b) }()
	return a, end, replies, ended
}

func TestAsksRunInOrderAndReply(t *testing.T) {
	a, _, replies, _ := servedAsks(t)
	for i := range 3 {
		a.ask(func(context.Context, *Bridge) tea.Msg { return i })
	}
	for want := range 3 {
		if got := <-replies; got != want {
			t.Errorf("reply %v, want %d", got, want)
		}
	}
}
