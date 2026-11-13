package tui

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"syscall"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/tui/internal/list"
)

// scanned stands for what a list leaves to the wait, of whatever type the list has.
type scanned struct {
	only    []string
	checked bool
}

func TestTheListHearsWhyAScanStopped(t *testing.T) {
	b, sent := testBridge(io.Discard)
	picks := make(chan scanned, 1)
	send := b.send
	b.send = func(msg tea.Msg) { send(msg); picks <- scanned{checked: true} }
	got, err := awaitPick(t.Context(), b, picks, func(context.Context, *Bridge) error { return errors.New("no network") })
	if err != nil || !got.checked {
		t.Errorf("awaitPick = %v, %v", got, err)
	}
	if !slices.Equal(*sent, []tea.Msg{list.ScannedMsg{Problem: "no network"}}) {
		t.Errorf("sent %v", *sent)
	}
}

func TestAChoiceEndsTheScanFirst(t *testing.T) {
	b, _ := testBridge(io.Discard)
	picks := make(chan scanned, 1)
	picks <- scanned{only: []string{"Keys"}}
	ended := false
	got, err := awaitPick(t.Context(), b, picks, func(ctx context.Context, b *Bridge) error {
		defer func() { ended = true }()
		return waitForCtx(ctx, b)
	})
	if err != nil || !slices.Equal(got.only, []string{"Keys"}) || !ended {
		t.Errorf("awaitPick = %v, %v, the scan ended: %v", got, err, ended)
	}
}

func TestAPanicOfTheScanOutlivesTheWait(t *testing.T) {
	b, _ := testBridge(io.Discard)
	picks := make(chan scanned, 1)
	picks <- scanned{}
	_, err := awaitPick(t.Context(), b, picks, func(ctx context.Context, _ *Bridge) error {
		<-ctx.Done()
		panic("the scan broke")
	})
	if !errors.Is(err, ErrPanic) {
		t.Errorf("awaitPick: %v", err)
	}
}

func TestABridgeReportReachesTheScreen(t *testing.T) {
	b, sent := testBridge(io.Discard)
	b.Report("Run", nil, "refused")
	if got := (*sent)[0].(ReportedMsg); got.Stage != "Run" || got.Problem != "refused" {
		t.Errorf("sent %+v", got)
	}
}

// A work past a choice ends well, whatever the cut.
func TestAWorkPastAChoiceEndsWellWhateverTheCut(t *testing.T) {
	c := noKeys(io.Discard)
	c.In, c.quiet = strings.NewReader("\x03"), true
	past := func(ctx context.Context, _ *Bridge) error { <-ctx.Done(); return nil }
	if err := Run(t.Context(), c, recorder{&[]tea.Msg{}}, past); !errors.Is(err, testCut(syscall.SIGINT)) {
		t.Errorf("Run: %v", err)
	}
}
