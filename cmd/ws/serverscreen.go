package main

import (
	"context"
	"fmt"
	"slices"
	"syscall"

	tea "charm.land/bubbletea/v2"
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
		return s.running(wizardOptions{check: check}, nil)
	}
	forms := func(context.Context) ([]tui.Sheet, func(context.Context), error) { return serverSheets(s.p), nil, nil }
	return tui.RunWizard(s.c.Context(), tui.Wizard{Title: s.c.CommandPath(), Config: s.config(), Round: s.round,
		Forms: forms})
}

// round is the works of a scan. The state screen reads ws on ws.
func (s serverScreen) round() tui.Round {
	return tui.Round{
		Stages: wizardQueue(s.stages(wizardOptions{})),
		Scan:   s.scan(),
		Read: func(tui.Side) (tui.StateReader, func(context.Context)) {
			return stateRead(serverOpen, s.p.version), nil
		},
		Run: func(_ context.Context, choice tui.QueueChoice) error { return s.chosen(choice) },
	}
}

// chosen runs the stages that choice holds, ws's run in part where the list left some of its roles out.
func (s serverScreen) chosen(choice tui.QueueChoice) error {
	name := wsRunOf(s.stages(wizardOptions{})).Name()
	o := wizardOptions{check: choice.Next == tui.NextCheck, update: slices.Contains(choice.Whole, name),
		only: choice.Partial[name], left: wizardLeft(choice)}
	return s.running(o, &choice)
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

// scan is a check of every stage, beside the list. ws speaks for its roles once its values are
// read, as the remote script's report takes ws_user from them.
func (s serverScreen) scan() tui.Work {
	return func(ctx context.Context, b *tui.Bridge) error {
		on, e := s.on(b)
		stages := serverWizardStages(wizardEffects{p: on, ask: b, talks: e.Talks, out: b, hand: &wizard.Hand{}},
			wizardOptions{check: true})
		run := wsRunOf(stages)
		watch := afterStage{b, run.WaitsOn(), func() {
			states, err := run.Ws.State(ctx, s.p.version)
			reportRoles(run.Name(), states, err, b.Report)
		}}
		o := b.Options(step.Options{Check: true})
		o.Observer = watch
		_, err := wizard.Run(ctx, on.machine, o, stages, s.p.lockDir(), 0, nil)
		return unwrapStep(err)
	}
}

// running runs o on the run screen of the stages' names, or on the list's, which choice holds. What
// stays by hand prints past the screen, which would paint over it.
func (s serverScreen) running(o wizardOptions, choice *tui.QueueChoice) error {
	cfg, done := runConfig(s.c, s.p, s.config())
	defer done()
	var screen tea.Model = tui.NewRunScreen(runTitle(s.c, o.check), cfg.Theme, tui.NewGroup("", s.stages(o)))
	if choice != nil {
		screen = tui.NewQueueRun(runTitle(s.c, o.check), cfg.Theme, *choice)
	}
	return tui.Run(s.c.Context(), cfg, screen, func(ctx context.Context, b *tui.Bridge) error {
		on, e := s.on(b)
		hand := &wizard.Hand{}
		stages := serverWizardStages(wizardEffects{p: on, ask: b, talks: e.Talks, out: b, events: b, hand: hand}, o)
		closing := func(ctx context.Context) { hand.Close(ctx, e.Text) }
		so := b.Options(step.Options{Check: o.check, LeftOut: o.left})
		_, err := wizard.Run(ctx, on.machine, so, stages, s.p.lockDir(), 0, closing)
		return unwrapStep(err)
	})
}
