package main

import (
	"bytes"
	"context"
	"slices"
	"strings"
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
