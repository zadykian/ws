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
	"charm.land/lipgloss/v2"
)

var esc = tea.KeyPressMsg{Code: tea.KeyEscape}

// haltedRunMsg is what a test's halt returns.
type haltedRunMsg struct{}

// haltingSession is a session whose work can stop between its steps.
func haltingSession() (session, *[]tea.Msg) {
	s, got := testSession()
	s.halt = func() tea.Msg { return haltedRunMsg{} }
	return s, got
}

// runs says whether cmd, a batch or not, gives want.
func runs(cmd tea.Cmd, want tea.Msg) bool {
	if cmd == nil {
		return false
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		return slices.ContainsFunc(batch, func(c tea.Cmd) bool { return runs(c, want) })
	}
	return msg == want
}

func TestSessionStopsPastTheCurrentStepOnceConfirmed(t *testing.T) {
	s, got := haltingSession()
	s, cmd := feed(s, ctrlC)
	if cmd != nil || !s.confirming || s.cutBy != nil {
		t.Fatalf("the first press stopped the run: cut by %v", s.cutBy)
	}
	if view := stripped(s.View().Content); !strings.Contains(view, "current step") || !strings.Contains(view, "[esc] go on") {
		t.Errorf("no question in:\n%s", view)
	}
	s, cmd = feed(s, enter)
	if !runs(cmd, haltedRunMsg{}) || !s.halted || s.confirming || s.stopping {
		t.Fatalf("halted %v, confirming %v, stopping %v", s.halted, s.confirming, s.stopping)
	}
	if !errors.Is(s.cutBy, cutError{syscall.SIGINT}) || !slices.Contains(*got, tea.Msg(haltedMsg{})) {
		t.Errorf("cut by %v, the screen got %v", s.cutBy, *got)
	}
	s, cmd = feed(s, ctrlC)
	if !runs(cmd, stoppedMsg{}) || !s.stopping {
		t.Fatalf("the second press left ctx running")
	}
	if _, cmd = feed(s, ctrlC); !quits(cmd) {
		t.Error("the third press left no way out")
	}
}

func TestSessionGoesOnPastEsc(t *testing.T) {
	s, _ := haltingSession()
	s, cmd := feed(s, ctrlC, esc)
	if cmd != nil || s.confirming || s.halted || s.cutBy != nil {
		t.Errorf("confirming %v, halted %v, cut by %v", s.confirming, s.halted, s.cutBy)
	}
	if s, _ = feed(s, ctrlC); !s.confirming {
		t.Error("a press past a refusal asks no more")
	}
}

func TestSessionWrapsItsNotesAndKeepsTheWayOut(t *testing.T) {
	s, _ := testSession()
	s, _ = feed(s, endedMsg{}, tea.WindowSizeMsg{Width: 24, Height: 8})
	got := stripped(s.View().Content)
	for _, want := range []string{"Ctrl-C again leaves it", "behind.", "current step."} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in:\n%s", want, got)
		}
	}
	s, _ = feed(s, tea.WindowSizeMsg{Width: 80, Height: 3})
	if view := s.View().Content; !strings.Contains(view, "Ctrl-C again") {
		t.Errorf("a short window hides the way out:\n%s", view)
	}
}

func TestSessionDrawsItsBoxInAnyWindow(t *testing.T) {
	s, _ := haltingSession()
	s, _ = feed(s, ctrlC)
	for w := range 14 {
		for h := range 10 {
			s, _ = feed(s, tea.WindowSizeMsg{Width: w, Height: h})
			if got := lipgloss.Width(s.View().Content); got > max(w, len("the screen")) {
				t.Errorf("%dx%d: drew %d columns", w, h, got)
			}
		}
	}
}

func TestBridgeStop(t *testing.T) {
	b := newBridge(io.Discard)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	stop := b.Stop(ctx)
	if err := stop(); err != nil {
		t.Errorf("before a stop: %v", err)
	}
	cancel()
	if err := stop(); !errors.Is(err, context.Canceled) {
		t.Errorf("past ctx's end: %v", err)
	}
	b.halted.Store(true)
	if err := stop(); !errors.Is(err, errHalted) {
		t.Errorf("past a stop: %v", err)
	}
}
