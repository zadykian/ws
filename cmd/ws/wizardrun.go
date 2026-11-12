package main

import (
	"context"
	"fmt"
	"slices"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/step"
)

// wizardRun is a wizard's run at a terminal, over its side: the Mac's or ws's.
type wizardRun struct {
	c    *cobra.Command
	p    process
	side wizardSide
	o    wizardOptions
}

// draw opens the lists over a scan and runs what they choose, or goes straight to the run under
// --check, as ws mac apply does.
func (w wizardRun) draw() error {
	if !opensList(w.c, w.p, false) {
		return w.whole()
	}
	return tui.RunWizard(w.c.Context(), tui.Wizard{Title: w.c.CommandPath(), Config: w.config(), Round: w.round,
		Forms: w.side.forms})
}

// round is the works of a scan. A new one reads the forms' answers. The state screen's read of ws and
// the run take the scan's sudo, which holds what it asked.
func (w wizardRun) round() tui.Round {
	sudo := &wizard.Sudo{}
	stages := w.plainStages()
	var run string // ws's run, found once a round; a side with none has no name to choose
	if r := wsRunOf(stages); r != nil {
		run = r.Name()
	}
	return tui.Round{
		Stages: wizardQueue(stages),
		Scan:   w.scan(sudo),
		Read:   func(side tui.Side) (tui.StateReader, func(context.Context)) { return w.side.read(side, sudo) },
		Run:    func(_ context.Context, choice tui.QueueChoice) error { return w.chosen(choice, run, sudo) },
	}
}

// config is what the wizard's screens draw with. A signal ends a plain wizard through its context,
// with status 1, so here too a signal is an error of the run, not signalExit.
func (w wizardRun) config() tui.Config {
	cfg := screenConfig(w.c, w.p)
	cfg.Cut = func(s syscall.Signal) error { return fmt.Errorf("stopped by %v", s) }
	return cfg
}

// plainStages returns the stages for their names and machines alone: none of them runs.
func (w wizardRun) plainStages() []step.Step {
	return w.side.stages(wizardEffects{p: w.p, hand: &wizard.Hand{}, sudo: &wizard.Sudo{}}, w.o)
}

// effects returns the stages' effects on b, whose stdout waits for the screen's end.
func (w wizardRun) effects(b *tui.Bridge, sudo *wizard.Sudo) wizardEffects {
	p, on := w.p, b.Effects()
	p.machine = on.M
	return wizardEffects{p: p, prog: w.c.CommandPath(), talks: on.Talks, ask: b, hidden: b, out: b, text: on.Text, obs: b,
		hand: &wizard.Hand{}, sudo: sudo, stop: b.Options(step.Options{}).Stop}
}

// scan is a check of every stage, beside the lists. ws speaks for its roles once the stage that
// installs it has read it, as ws's stages behind that need a ws at this version.
func (w wizardRun) scan(sudo *wizard.Sudo) tui.Work {
	check := w.o
	check.check = true
	return func(ctx context.Context, b *tui.Bridge) error {
		e := w.effects(b, sudo)
		stages := w.side.stages(e, check)
		o := b.Options(step.Options{Check: true})
		if run := wsRunOf(stages); run != nil {
			o.Observer = afterStage{b, run.WaitsOn(), func() {
				states, err := run.Ws.State(ctx, w.p.version)
				reportRoles(run.Name(), states, err, b.Report)
			}}
		}
		_, err := wizard.Run(ctx, e.p.machine, o, stages, w.p.lockDir(), w.side.pid, nil)
		return unwrapStep(err)
	}
}

// afterStage passes a run's events to next, and calls then once the stage named has passed. A step
// of a stage's own has a parent, and a stage skipped has not read what then needs.
type afterStage struct {
	next  step.Observer
	stage string
	then  func()
}

// Observe passes e on, ahead of then, as the lists take the stage's result first.
func (a afterStage) Observe(e step.Event) {
	a.next.Observe(e)
	if e.Parent == "" && e.Step == a.stage && e.Kind == step.Done {
		a.then()
	}
}

// wsRunOf returns the stage of ws's run, or nil where the stages hold none.
func wsRunOf(stages []step.Step) *wizard.WsRun {
	for _, s := range stages {
		if run, ok := s.(*wizard.WsRun); ok {
			return run
		}
	}
	return nil
}

// work is a run of the stages by o beside a screen. What stays by hand goes to the run's stdout, as in
// a plain run.
func (w wizardRun) work(o wizardOptions, sudo *wizard.Sudo) tui.Work {
	return func(ctx context.Context, b *tui.Bridge) error {
		e := w.effects(b, sudo)
		e.events = b
		closing := func(ctx context.Context) { e.hand.Close(ctx, e.text) }
		so := b.Options(step.Options{Check: o.check, LeftOut: o.left})
		_, err := wizard.Run(ctx, e.p.machine, so, w.side.stages(e, o), w.p.lockDir(), w.side.pid, closing)
		return unwrapStep(err)
	}
}

// running gives draw a run's config and the work of o. The run's stdout prints past the screen,
// which it would paint over.
func (w wizardRun) running(o wizardOptions, sudo *wizard.Sudo, draw func(tui.Config, tui.Work) error) error {
	cfg, done := runConfig(w.c, w.p, w.config())
	defer done()
	return draw(cfg, w.work(o, sudo))
}

// whole runs every stage, on the run screen of their names.
func (w wizardRun) whole() error {
	return w.running(w.o, &wizard.Sudo{}, func(cfg tui.Config, work tui.Work) error {
		screen := tui.NewRunScreen(runTitle(w.c, w.o.check), cfg.Theme, tui.NewGroup("", w.plainStages()))
		return tui.Run(w.c.Context(), cfg, screen, work)
	})
}

// optionsOf is the options of a run of what the lists chose, run being ws's run's name. A stage in part leaves out ws's roles by
// --only, as they run on ws, and ws mac apply's steps by the run's filter. ws's run whole by a tick goes
// once more, as under --update: a good last run says nothing of a step that drifted after it.
func (w wizardRun) optionsOf(choice tui.QueueChoice, run string) wizardOptions {
	o := w.o
	o.check = choice.Next == tui.NextCheck
	o.update = o.update || slices.Contains(choice.Whole, run)
	o.left, o.only = wizardLeft(choice), choice.Partial[run]
	return o
}

// chosen runs what the lists chose, on a run of those stages.
func (w wizardRun) chosen(choice tui.QueueChoice, run string, sudo *wizard.Sudo) error {
	o := w.optionsOf(choice, run)
	return w.running(o, sudo, func(cfg tui.Config, work tui.Work) error {
		screen := tui.NewQueueRun(runTitle(w.c, o.check), cfg.Theme, choice)
		return tui.Run(w.c.Context(), cfg, screen, work)
	})
}
