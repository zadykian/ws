package tui

import (
	"context"
	"errors"
	"io"
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/tui/internal/list"
)

func TestTheListHearsWhyAScanStopped(t *testing.T) {
	b, sent := testBridge(io.Discard)
	picks := make(chan QueueChoice, 1)
	send := b.send
	b.send = func(msg tea.Msg) { send(msg); picks <- QueueChoice{Next: NextCheck} }
	got, err := awaitPick(t.Context(), b, picks, func(context.Context, *Bridge) error { return errors.New("no network") })
	if err != nil || got.Next != NextCheck {
		t.Errorf("awaitPick = %v, %v", got, err)
	}
	if !slices.Equal(*sent, []tea.Msg{list.ScannedMsg{Problem: "no network"}}) {
		t.Errorf("sent %v", *sent)
	}
}

func TestAChoiceEndsTheScanFirst(t *testing.T) {
	b, _ := testBridge(io.Discard)
	picks := make(chan QueueChoice, 1)
	picks <- QueueChoice{Only: []string{"Keys"}}
	ended := false
	got, err := awaitPick(t.Context(), b, picks, func(ctx context.Context, b *Bridge) error {
		defer func() { ended = true }()
		return waitForCtx(ctx, b)
	})
	if err != nil || !slices.Equal(got.Only, []string{"Keys"}) || !ended {
		t.Errorf("awaitPick = %v, %v, the scan ended: %v", got, err, ended)
	}
}

func TestAPanicOfTheScanOutlivesTheWait(t *testing.T) {
	b, _ := testBridge(io.Discard)
	picks := make(chan QueueChoice, 1)
	picks <- QueueChoice{}
	_, err := awaitPick(t.Context(), b, picks, func(ctx context.Context, _ *Bridge) error {
		<-ctx.Done()
		panic("the scan broke")
	})
	if !errors.Is(err, errPanic) {
		t.Errorf("awaitPick: %v", err)
	}
}
