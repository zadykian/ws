package tui

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"syscall"
	"testing"

	tea "charm.land/bubbletea/v2"
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
	if !slices.Equal(*sent, []tea.Msg{ScannedMsg{"no network"}}) {
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

func TestPickLeavesNothingPastTheScreen(t *testing.T) {
	var out bytes.Buffer
	c := noKeys(&out)
	c.In = strings.NewReader("q")
	got, err := Pick(t.Context(), c, "ws", nil, func(ctx context.Context, b *Bridge) error {
		b.Warn("a line of the scan")
		b.ByHand("a line of the scan")
		return waitForCtx(ctx, b)
	})
	if err != nil || got.Only != nil {
		t.Errorf("Pick = %+v, %v", got, err)
	}
	left := out.String()[strings.LastIndex(out.String(), leaveBuffer):]
	if strings.Contains(left, "a line of the scan") {
		t.Errorf("past the screen: %q", left)
	}
}

func TestPickEndsInTheCutOfCtrlC(t *testing.T) {
	c := noKeys(io.Discard)
	c.In = strings.NewReader("\x03")
	if _, err := Pick(t.Context(), c, "ws", nil, waitForCtx); !errors.Is(err, testCut(syscall.SIGINT)) {
		t.Errorf("Pick: %v", err)
	}
	c.In, c.quiet = strings.NewReader("\x03"), true // A work past a choice ends well, whatever the cut.
	past := func(ctx context.Context, _ *Bridge) error { <-ctx.Done(); return nil }
	if err := Run(t.Context(), c, recorder{&[]tea.Msg{}}, past); !errors.Is(err, testCut(syscall.SIGINT)) {
		t.Errorf("Run: %v", err)
	}
}
