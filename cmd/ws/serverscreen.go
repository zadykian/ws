package main

import (
	"bytes"
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

// serverScreen is ws server wizard at a terminal. tmp is where the lock of a run goes.
type serverScreen struct {
	c   *cobra.Command
	p   process
	tmp string
}

// draw opens the list of the stages over a scan and runs what it chooses, or goes straight to the
// run under --check. Every stage runs on ws, so the list is ws's alone.
func (s serverScreen) draw(check bool) error {
	if !opensList(s.c, s.p, false) {
		return s.running(serverPick{check: check}, nil)
	}
	forms := func(context.Context) ([]tui.Sheet, func(context.Context), error) { return serverSheets(s.p), nil, nil }
	return tui.RunWizard(s.c.Context(), tui.Wizard{Title: s.c.CommandPath(), Config: s.config(), Round: s.round,
		Forms: forms})
}

// round is the works of a scan. The state screen reads ws on ws.
func (s serverScreen) round() tui.Round {
	return tui.Round{
		Stages: wizardQueue(s.p, s.stages(serverPick{})),
		Scan:   s.scan(),
		Read: func(tui.Side) (tui.StateReader, func(context.Context)) {
			return stateRead(serverOpen, s.p.version), nil
		},
		Run: func(_ context.Context, choice tui.Choice) error { return s.chosen(choice) },
	}
}

// chosen runs the stages that choice holds, ws's run in part where the list left some of its roles out.
func (s serverScreen) chosen(choice tui.Choice) error {
	name := wsRunOf(s.stages(serverPick{})).Name()
	r := serverPick{check: choice.Check, update: slices.Contains(choice.Whole, name), only: choice.Partial[name]}
	return s.running(r, &choice)
}

// config is what the screens draw with. A signal ends a plain run through its context, with status
// 1, so here too a signal is the run's error.
func (s serverScreen) config() tui.Config {
	cfg := screenConfig(s.c, s.p)
	cfg.Cut = func(sig syscall.Signal) error { return fmt.Errorf("stopped by %v", sig) }
	return cfg
}

// stages returns the stages for their names and machines alone: none of them runs.
func (s serverScreen) stages(r serverPick) []step.Step {
	return serverWizardStages(s.p, nil, nil, nil, nil, &wizard.Hand{}, r)
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
		stages := serverWizardStages(on, b, e.Talks, b, nil, &wizard.Hand{}, serverPick{check: true})
		run := wsRunOf(stages)
		watch := afterStage{b, run.WaitsOn(), func() {
			states, err := run.Ws.State(ctx, s.p.version)
			reportRoles(run.Name(), states, err, b.Report)
		}}
		o := b.Options(step.Options{Check: true})
		o.Observer = watch
		_, err := wizard.Run(ctx, on.machine, o, stages, s.tmp, 0, nil)
		return unwrapStep(err)
	}
}

// running runs r on the run screen of the stages' names, or on the list's, which choice holds. What
// stays by hand prints past the screen, which would paint over it.
func (s serverScreen) running(r serverPick, choice *tui.Choice) error {
	cfg, done := runConfig(s.c, s.p, s.config())
	defer done()
	var screen tea.Model = tui.NewRunScreen(runTitle(s.c, r.check), cfg.Theme, tui.NewGroup("", s.stages(r)))
	o := step.Options{Check: r.check}
	if choice != nil {
		screen, o.LeftOut = tui.NewQueueRun(runTitle(s.c, r.check), cfg.Theme, *choice), wizardLeft(*choice, nil)
	}
	var text bytes.Buffer
	err := tui.Run(s.c.Context(), cfg, screen, func(ctx context.Context, b *tui.Bridge) error {
		on, e := s.on(b)
		hand := &wizard.Hand{}
		stages := serverWizardStages(on, b, e.Talks, b, b, hand, r)
		closing := func(ctx context.Context) { hand.Close(ctx, &text) }
		o = b.Options(o)
		_, err := wizard.Run(ctx, on.machine, o, stages, s.tmp, 0, closing)
		b.Log(text.String()) // A failed run prints it too.
		return unwrapStep(err)
	})
	if err == nil { // A work left behind may still write to text.
		_, _ = text.WriteTo(s.c.OutOrStdout())
	}
	return err
}
