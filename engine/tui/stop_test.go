package tui

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// haltedRunMsg is what a test's halt returns.
type haltedRunMsg struct{}

// haltingSession is a session whose work can stop between its steps.
func haltingSession() (session, *[]tea.Msg) {
	s, got := testSession()
	s.halt = func() tea.Msg { return haltedRunMsg{} }
	s.took = new(atomic.Bool)
	s.took.Store(true)
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
	if !errors.Is(s.cutBy, testCut(syscall.SIGINT)) || !slices.Contains(*got, tea.Msg(haltedMsg{})) {
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

func TestSessionStopButtonGoesOnByTheFocus(t *testing.T) {
	s, _ := haltingSession()
	if s, _ = feed(s, stopPressedMsg{}, tab, tab, enter); s.confirming || s.halted {
		t.Errorf("Enter on the second button: confirming %v, halted %v", s.confirming, s.halted)
	}
}

func TestSessionEndsCtxWhereNothingConfirms(t *testing.T) {
	ask := askMsg{prompt{id: 1, text: "Host"}}
	cases := []struct {
		name string
		msgs []tea.Msg
	}{
		{"a second press", []tea.Msg{ctrlC, ctrlC}},
		{"a signal", []tea.Msg{signalMsg{syscall.SIGINT}}},
		{"a signal under the box", []tea.Msg{ctrlC, signalMsg{syscall.SIGINT}}},
		{"a press under a question", []tea.Msg{ask, ctrlC}},
		{"a press past ctx's end", []tea.Msg{endedMsg{}, ctrlC}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s, _ := haltingSession()
			s, cmd := feed(s, c.msgs...)
			if !runs(cmd, stoppedMsg{}) || !s.stopping || s.confirming {
				t.Errorf("stopping %v, confirming %v", s.stopping, s.confirming)
			}
		})
	}
}

func TestSessionWithNoHaltEndsCtxOnTheFirstPress(t *testing.T) {
	s, _ := testSession()
	if s, cmd := feed(s, stopPressedMsg{}); !runs(cmd, stoppedMsg{}) || s.confirming {
		t.Errorf("confirming %v", s.confirming)
	}
}

func TestSessionGivesAQuestionTheKeysOverTheBox(t *testing.T) {
	s, _ := haltingSession()
	s, _ = feed(s, ctrlC, askMsg{prompt{id: 1, text: "Host"}}, press('a'), enter)
	if s.halted || !s.confirming {
		t.Errorf("the answer's Enter reached the box: halted %v, confirming %v", s.halted, s.confirming)
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
	stop := b.stop(ctx)
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

// keysAfterOptions returns a Config whose keys come once the work holds its options: a reader
// that is ready at once races the work's start. The pipe closes with the test.
func keysAfterOptions(t *testing.T) (Config, func()) {
	r, w := io.Pipe()
	t.Cleanup(func() { _ = w.Close() })
	c := noKeys(io.Discard)
	c.In = r
	return c, func() { _, _ = io.WriteString(w, "\x03\r") }
}

func TestRunStopsPastTheCurrentStepWithCtxAlive(t *testing.T) {
	c, press := keysAfterOptions(t)
	alive := false
	err := Run(t.Context(), c, recorder{&[]tea.Msg{}}, func(ctx context.Context, b *Bridge) error {
		stop := b.Options(step.Options{}).Stop
		press()
		for stop() == nil {
			time.Sleep(time.Millisecond)
		}
		alive = ctx.Err() == nil
		return stop()
	})
	if !errors.Is(err, testCut(syscall.SIGINT)) || !alive {
		t.Errorf("Run: %v, ctx alive %v", err, alive)
	}
}

func TestRunEndsWellWhereTheStoppedStepWasTheLast(t *testing.T) {
	c, press := keysAfterOptions(t)
	err := Run(t.Context(), c, recorder{&[]tea.Msg{}}, func(_ context.Context, b *Bridge) error {
		stop := b.Options(step.Options{}).Stop
		press()
		for stop() == nil {
			time.Sleep(time.Millisecond)
		}
		return nil
	})
	if err != nil {
		t.Errorf("Run: %v", err)
	}
}

// Before a work took Bridge.Options it reads no stop, so Ctrl-C ends ctx at once.
func TestCtrlCEndsTheRunBeforeTheWorkTookTheOptions(t *testing.T) {
	c := noKeys(io.Discard)
	c.In = strings.NewReader("\x03")
	err := Run(t.Context(), c, recorder{&[]tea.Msg{}}, waitForCtx)
	if !errors.Is(err, testCut(syscall.SIGINT)) {
		t.Errorf("Run: %v", err)
	}
}

func TestRunScreenStopButtonAsksTheSession(t *testing.T) {
	r := testRun()
	for range 4 {
		r = sent(r, tab)
	}
	if _, cmd := r.Update(enter); !runs(cmd, stopPressedMsg{}) {
		t.Error("the button asked nothing")
	}
}

func TestRunScreenShowsAStopInItsWindowsTitle(t *testing.T) {
	r := sent(testRun(), haltedMsg{}, widget.Line{Kind: widget.Piped, Text: "still printing"})
	got := shown(r)
	for _, want := range []string{"Stopping after base: Packages", "still printing", "[ctrl+c] stop now"} {
		if !strings.Contains(got, want) {
			t.Errorf("frame holds no %q:\n%s", want, got)
		}
	}
}
