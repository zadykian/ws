package main

import (
	"bytes"
	"context"
	"errors"
	"slices"
	"strings"
	"syscall"
	"testing"

	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
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
