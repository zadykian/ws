package tui

import (
	"context"
	"errors"
	"io"
	"slices"
	"syscall"
	"testing"

	tea "charm.land/bubbletea/v2"
)

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

func TestSessionStopsPastTheCurrentStep(t *testing.T) {
	s, got := haltingSession()
	s, cmd := feed(s, ctrlC)
	if !runs(cmd, haltedRunMsg{}) || !s.halted || s.stopping {
		t.Fatalf("halted %v, stopping %v", s.halted, s.stopping)
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
