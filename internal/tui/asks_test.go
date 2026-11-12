package tui

import (
	"context"
	"errors"
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

func TestAsksLeaveEndsARunningAskAndServeReturnsNil(t *testing.T) {
	a, _, _, ended := servedAsks(t)
	running := make(chan struct{})
	a.ask(func(ctx context.Context, _ *Bridge) tea.Msg {
		close(running)
		<-ctx.Done()
		return nil
	})
	<-running
	if msg := a.leave(); msg != nil {
		t.Errorf("leave returned %v", msg)
	}
	if err := <-ended; err != nil {
		t.Errorf("serve returned %v, want nil", err)
	}
}

func TestAsksAskAfterTheEndReturnsAtOnce(t *testing.T) {
	a, _, _, ended := servedAsks(t)
	a.leave()
	if err := <-ended; err != nil {
		t.Error(err)
	}
	a.ask(func(context.Context, *Bridge) tea.Msg { return nil })
}

func TestAsksCutReturnsCtxError(t *testing.T) {
	a, end, _, ended := servedAsks(t)
	end(context.Canceled)
	if err := <-ended; !errors.Is(err, context.Canceled) {
		t.Errorf("serve returned %v, want the cut", err)
	}
	a.ask(func(context.Context, *Bridge) tea.Msg { return nil })
}

func TestAsksDropAReplyPastCtxEnd(t *testing.T) {
	a, _, replies, ended := servedAsks(t)
	a.ask(func(_ context.Context, _ *Bridge) tea.Msg {
		a.leave()
		return "late"
	})
	if err := <-ended; err != nil {
		t.Error(err)
	}
	select {
	case msg := <-replies:
		t.Errorf("sent %v past the end", msg)
	default:
	}
}
