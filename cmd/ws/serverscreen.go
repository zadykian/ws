package main

import (
	"context"
	"fmt"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/step"
)

// serverScreen is ws server wizard at a terminal.
type serverScreen struct {
	c *cobra.Command
	p process
}

// draw opens the list of the stages over a scan and runs what it chooses, or goes straight to the
// run under --check. Every stage runs on ws, so the list is ws's alone.
func (s serverScreen) draw(check bool) error {
	if !opensList(s.c, s.p, false) {
		return s.running(wizardOptions{check: check})
	}
	return wizardRun{s.c, s.p, serverSide(s.p), wizardOptions{}}.draw()
}

// config is what the screens draw with. A signal ends a plain run through its context, with status
// 1, so here too a signal is the run's error.
func (s serverScreen) config() tui.Config {
	cfg := screenConfig(s.c, s.p)
	cfg.Cut = func(sig syscall.Signal) error { return fmt.Errorf("stopped by %v", sig) }
	return cfg
}

// stages returns the stages for their names and machines alone: none of them runs.
func (s serverScreen) stages(o wizardOptions) []step.Step {
	return serverWizardStages(wizardEffects{p: s.p, hand: &wizard.Hand{}}, o)
}

// on returns the process and the screen's effects as a run beside the screen takes them.
func (s serverScreen) on(b *tui.Bridge) (process, tui.Effects) {
	on, e := s.p, b.Effects()
	on.machine = e.M
	return on, e
}

// running runs o on the run screen of the stages' names. What stays by hand prints past the screen,
// which would paint over it.
func (s serverScreen) running(o wizardOptions) error {
	cfg, done := runConfig(s.c, s.p, s.config())
	defer done()
	screen := tui.NewRunScreen(runTitle(s.c, o.check), cfg.Theme, tui.NewGroup("", s.stages(o)))
	return tui.Run(s.c.Context(), cfg, screen, func(ctx context.Context, b *tui.Bridge) error {
		on, e := s.on(b)
		hand := &wizard.Hand{}
		stages := serverWizardStages(wizardEffects{p: on, ask: b, talks: e.Talks, out: b, events: b, hand: hand}, o)
		closing := func(ctx context.Context) { hand.Close(ctx, e.Text) }
		so := b.Options(step.Options{Check: o.check})
		_, err := wizard.Run(ctx, on.machine, so, stages, s.p.lockDir(), 0, closing)
		return unwrapStep(err)
	})
}
