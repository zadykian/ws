package main

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/step"
)

// macScreen is ws mac wizard at a terminal.
type macScreen struct {
	c *cobra.Command
	p process
	o wizardOptions
}

// draw opens the lists over a scan and runs what they choose, or goes straight to the run under
// --check, as ws mac apply does.
func (w macScreen) draw() error {
	if !opensList(w.c, w.p, false) {
		return w.whole()
	}
	return tui.RunWizard(w.c.Context(), tui.Wizard{Title: w.c.CommandPath(), Config: w.config(), Round: w.round,
		Forms: macForms(w.c, w.p)})
}

// round is the works of a scan. A new one reads the forms' answers. The state screen's read of ws and
// the run take the scan's sudo, which holds what it asked.
func (w macScreen) round() tui.Round {
	sudo := &wizard.Sudo{}
	return tui.Round{
		Stages: wizardQueue(w.plainStages()),
		Scan:   w.scan(sudo),
		Read:   func(side tui.Side) (tui.StateReader, func(context.Context)) { return w.read(side, sudo) },
		Run:    func(_ context.Context, choice tui.Choice) error { return w.chosen(choice, sudo) },
	}
}

// config is what the wizard's screens draw with. A signal ends a plain wizard through its context,
// with status 1, so here too a signal is an error of the run, not signalExit.
func (w macScreen) config() tui.Config {
	cfg := screenConfig(w.c, w.p)
	cfg.Cut = func(s syscall.Signal) error { return fmt.Errorf("stopped by %v", s) }
	return cfg
}

// plainStages returns the stages for their names and machines alone: none of them runs.
func (w macScreen) plainStages() []step.Step {
	return wizardStages(wizardEffects{p: w.p, hand: &wizard.Hand{}, sudo: &wizard.Sudo{}}, w.o)
}

// effects returns the stages' effects on b, whose stdout waits for the screen's end.
func (w macScreen) effects(b *tui.Bridge, sudo *wizard.Sudo) wizardEffects {
	p, on := w.p, b.Effects()
	p.machine = on.M
	return wizardEffects{p: p, prog: w.c.CommandPath(), talks: on.Talks, ask: b, hidden: b, out: b, text: on.Text, obs: b,
		hand: &wizard.Hand{}, sudo: sudo, stop: b.Options(step.Options{}).Stop}
}

// scan is a check of every stage, beside the lists. ws speaks for its roles once the stage that
// installs it has read it, as ws's stages behind that need a ws at this version.
func (w macScreen) scan(sudo *wizard.Sudo) tui.Work {
	check := w.o
	check.check = true
	return func(ctx context.Context, b *tui.Bridge) error {
		e := w.effects(b, sudo)
		stages := wizardStages(e, check)
		run := wsRunOf(stages)
		watch := afterStage{b, run.WaitsOn(), func() {
			states, err := run.Ws.State(ctx, w.p.version)
			reportRoles(run.Name(), states, err, b.Report)
		}}
		o := b.Options(step.Options{Check: true})
		o.Observer = watch
		_, err := wizard.Run(ctx, e.p.machine, o, stages, w.tmp(), w.p.pid, nil)
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

// wsRunOf returns the stage of ws's run.
func wsRunOf(stages []step.Step) *wizard.WsRun {
	for _, s := range stages {
		if run, ok := s.(*wizard.WsRun); ok {
			return run
		}
	}
	panic("the wizard has no stage of ws's run")
}

// tmp is where the lock of a run goes.
func (w macScreen) tmp() string { return cmp.Or(w.p.getenv("TMPDIR"), "/tmp") }

// wizardRun is what a run of the stages takes from a screen.
type wizardRun struct {
	o    wizardOptions
	sudo *wizard.Sudo
	// left and only are a choice's. Nil ones run every stage whole.
	left func(parent, name string) bool
	only []string
}

// work is r beside a screen. What stays by hand goes to the run's stdout, as in a plain run.
func (w macScreen) work(r wizardRun) tui.Work {
	return func(ctx context.Context, b *tui.Bridge) error {
		e := w.effects(b, r.sudo)
		e.left, e.only, e.events = r.left, r.only, b
		closing := func(ctx context.Context) { e.hand.Close(ctx, e.text) }
		o := b.Options(step.Options{Check: r.o.check, LeftOut: r.left})
		_, err := wizard.Run(ctx, e.p.machine, o, wizardStages(e, r.o), w.tmp(), w.p.pid, closing)
		return unwrapStep(err)
	}
}

// running gives draw a run's config and the work of r. The run's stdout prints past the screen,
// which it would paint over.
func (w macScreen) running(r wizardRun, draw func(tui.Config, tui.Work) error) error {
	cfg, done := runConfig(w.c, w.p, w.config())
	defer done()
	return draw(cfg, w.work(r))
}

// whole runs every stage, on the run screen of their names.
func (w macScreen) whole() error {
	return w.running(wizardRun{o: w.o, sudo: &wizard.Sudo{}}, func(cfg tui.Config, work tui.Work) error {
		screen := tui.NewRunScreen(runTitle(w.c, w.o.check), cfg.Theme, tui.NewGroup("", w.plainStages()))
		return tui.Run(w.c.Context(), cfg, screen, work)
	})
}

// runOf is the run of what the lists chose. A stage in part leaves out ws's roles by --only, as they
// run on ws, and ws mac apply's steps by the run's filter. ws's run whole by a tick goes once more, as
// under --update: a good last run says nothing of a step that drifted after it.
func (w macScreen) runOf(choice tui.Choice, sudo *wizard.Sudo) wizardRun {
	o := w.o
	o.check = choice.Check
	name := wsRunOf(w.plainStages()).Name()
	o.update = o.update || slices.Contains(choice.Whole, name)
	return wizardRun{o: o, sudo: sudo, left: wizardLeft(choice, setup.Names()), only: choice.Partial[name]}
}

// chosen runs what the lists chose, on a run of those stages.
func (w macScreen) chosen(choice tui.Choice, sudo *wizard.Sudo) error {
	r := w.runOf(choice, sudo)
	return w.running(r, func(cfg tui.Config, work tui.Work) error {
		screen := tui.NewQueueRun(runTitle(w.c, r.o.check), cfg.Theme, choice)
		return tui.Run(w.c.Context(), cfg, screen, work)
	})
}
