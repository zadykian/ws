package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"syscall"
	"testing"

	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/step"
)

func TestWizardWorkRunsTheStagesOnTheScreensEffects(t *testing.T) {
	p, m := macTerminal()
	c, _ := screenCommand(t, "")
	w := wizardScreen{c, p, wizardOptions{}}
	r := wizardRun{o: wizardOptions{check: true}, sudo: &wizard.Sudo{}}
	cfg := w.config()
	screen := tui.NewRunScreen("ws", cfg.Theme, tui.Group{})
	var text bytes.Buffer
	if err := tui.Run(context.Background(), cfg, screen, w.work(r, &text)); err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "run caffeinate") }) {
		t.Errorf("no run began: calls = %q", m.Calls)
	}
}

func TestWizardConfigEndsACutAsAPlainRunDoes(t *testing.T) {
	p, _ := macTerminal()
	c, _ := screenCommand(t, "")
	err := wizardScreen{c, p, wizardOptions{}}.config().Cut(syscall.SIGINT)
	var cut signalExit
	if err == nil || errors.As(err, &cut) {
		t.Errorf("err = %v, want one of status 1", err)
	}
}

func TestWizardCheckAtATerminalGoesStraightToTheRun(t *testing.T) {
	p, m := macTerminal()
	_, stdout, stderr := runWs(p, "", "mac", "wizard", "--check")
	if stdout != "" || !strings.Contains(stderr, enterScreen) || !strings.Contains(stderr, leaveScreen) {
		t.Errorf("stdout = %q, stderr = %q", stdout, stderr)
	}
	if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "run caffeinate") }) {
		t.Errorf("no run began: calls = %q", m.Calls)
	}
}

func TestWizardPlainDrawsNoScreenAtATerminal(t *testing.T) {
	p, _ := macTerminal()
	_, _, stderr := runWs(p, "", "mac", "wizard", "--plain", "--check")
	if strings.Contains(stderr, enterScreen) {
		t.Errorf("--plain drew: %q", stderr)
	}
}

func TestWizardWorkLeavesOutWhatTheChoiceDoes(t *testing.T) {
	p, m := macTerminal()
	c, _ := screenCommand(t, "")
	w := wizardScreen{c, p, wizardOptions{}}
	r := wizardRun{sudo: &wizard.Sudo{}, left: func(string, string) bool { return true }}
	cfg := w.config()
	screen := tui.NewRunScreen("ws", cfg.Theme, tui.Group{})
	if err := tui.Run(context.Background(), cfg, screen, w.work(r, io.Discard)); err != nil {
		t.Fatal(err)
	}
	if slices.Contains(m.Calls, "run xcode-select --install") {
		t.Errorf("a stage left out applied: calls = %q", m.Calls)
	}
}

func TestWizardScanChangesNothing(t *testing.T) {
	p, m := macTerminal()
	c, _ := screenCommand(t, "")
	w := wizardScreen{c, p, wizardOptions{}}
	cfg := w.config()
	screen := tui.NewRunScreen("ws", cfg.Theme, tui.Group{})
	if err := tui.Run(context.Background(), cfg, screen, w.scan(&wizard.Sudo{})); err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(m.Calls, "run xcode-select -p") || slices.Contains(m.Calls, "run xcode-select --install") {
		t.Errorf("calls = %q, want a check of the tools alone", m.Calls)
	}
}

func TestAfterStageWaitsForTheStageItself(t *testing.T) {
	var heard, called []string
	a := afterStage{observer(func(e step.Event) { heard = append(heard, e.Step) }), "one", func() {
		called = append(called, heard[len(heard)-1])
	}}
	a.Observe(step.Event{Step: "one", Kind: step.Started})
	a.Observe(step.Event{Step: "one", Kind: step.Skipped})
	a.Observe(step.Event{Parent: "two", Step: "one", Kind: step.Done})
	a.Observe(step.Event{Step: "two", Kind: step.Done})
	if len(called) != 0 {
		t.Fatalf("then ran early: %q", called)
	}
	a.Observe(step.Event{Step: "one", Kind: step.Done})
	if !slices.Equal(called, []string{"one"}) {
		t.Errorf("then = %q, want it once, past the event", called)
	}
}

// observer hears a run's events through a function.
type observer func(step.Event)

func (f observer) Observe(e step.Event) { f(e) }
