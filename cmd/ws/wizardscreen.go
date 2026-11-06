package main

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"io"
	"slices"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/step"
)

// wizardScreen is ws mac wizard at a terminal.
type wizardScreen struct {
	c *cobra.Command
	p process
	o wizardOptions
}

// draw opens the lists over a scan and runs what they choose, or goes straight to the run under
// --check, as ws mac apply does. The forms come back to a new scan, of what they wrote, and the
// state screen, which writes nothing, to the lists as it found them.
func (w wizardScreen) draw() error {
	if !opensList(w.c, w.p, false) {
		return w.whole()
	}
	for {
		// A new one reads the forms' answers. The run takes the scan's, which holds what it asked.
		sudo := &wizard.Sudo{}
		queue := wizardQueue(w.p, w.plainStages())
		choice, err := tui.PickStages(w.c.Context(), w.config(), w.c.CommandPath(), queue, w.scan(sudo))
		for err == nil && choice.State {
			choice, err = w.stated(choice, sudo)
		}
		switch {
		case err != nil:
			return err
		case choice.Configure:
			if err := configureOn(w.c, w.p, w.config()); err != nil {
				return err
			}
		case len(choice.Only) == 0:
			return nil
		default:
			return w.chosen(choice, sudo)
		}
	}
}

// config is what the wizard's screens draw with. A signal ends a plain wizard through its context,
// with status 1, so here too it is an error of the run, not signalExit.
func (w wizardScreen) config() tui.Config {
	cfg := screenConfig(w.c, w.p)
	cfg.Cut = func(s syscall.Signal) error { return fmt.Errorf("stopped by %v", s) }
	return cfg
}

// plainStages returns the stages for their names and machines alone: none of them runs.
func (w wizardScreen) plainStages() []step.Step {
	return wizardStages(wizardEffects{p: w.p, hand: &wizard.Hand{}, sudo: &wizard.Sudo{}}, w.o)
}

// effects returns the stages' effects on b, for a run that prints its stdout on text.
func (w wizardScreen) effects(ctx context.Context, b *tui.Bridge, text io.Writer, sudo *wizard.Sudo) wizardEffects {
	p := w.p
	p.machine = screenMachine(ctx, b, p.machine)
	talks, _ := p.machine.(effect.Foreground)
	if m, ok := p.machine.(effect.OSMachine); ok {
		talks = b.Foreground(m)
	}
	return wizardEffects{p: p, prog: w.c.CommandPath(), talks: talks, ask: b, hidden: b, out: b, text: text, obs: b,
		hand: &wizard.Hand{}, sudo: sudo, stop: b.Stop(ctx)}
}

// scan is a check of every stage, beside the lists. ws speaks for its roles once the stage that
// installs it has read it, as ws's stages behind that need a ws at this version.
func (w wizardScreen) scan(sudo *wizard.Sudo) tui.Work {
	check := w.o
	check.check = true
	return func(ctx context.Context, b *tui.Bridge) error {
		e := w.effects(ctx, b, io.Discard, sudo)
		stages := wizardStages(e, check)
		run := wsRunOf(stages)
		watch := afterStage{b, run.WaitsOn(), func() {
			states, err := run.Remote.State(ctx, w.p.version)
			reportRoles(run.Name(), states, err, b.Report)
		}}
		o := step.Options{Observer: watch, Check: true, Stop: e.stop}
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
func (w wizardScreen) tmp() string { return cmp.Or(w.p.getenv("TMPDIR"), "/tmp") }

// wizardRun is what a run of the stages takes from a screen.
type wizardRun struct {
	o    wizardOptions
	sudo *wizard.Sudo
	// left and only are a choice's. Nil ones run every stage whole.
	left func(parent, name string) bool
	only []string
}

// work is r beside a screen. What stays by hand goes to the run's stdout, as in a plain run.
func (w wizardScreen) work(r wizardRun, text io.Writer) tui.Work {
	return func(ctx context.Context, b *tui.Bridge) error {
		e := w.effects(ctx, b, text, r.sudo)
		e.left, e.only, e.events = r.left, r.only, b
		closing := func(ctx context.Context) { e.hand.Close(ctx, text) }
		o := step.Options{Observer: b, Check: r.o.check, Stop: e.stop, LeftOut: r.left}
		_, err := wizard.Run(ctx, e.p.machine, o, wizardStages(e, r.o), w.tmp(), w.p.pid, closing)
		return unwrapStep(err)
	}
}

// running gives draw a run's config and the work of r, and prints the run's stdout past the screen,
// which it would paint over.
func (w wizardScreen) running(r wizardRun, draw func(tui.Config, tui.Work) error) error {
	cfg, done := runConfig(w.c, w.p, w.config())
	defer done()
	var text bytes.Buffer
	work := w.work(r, &text)
	err := draw(cfg, func(ctx context.Context, b *tui.Bridge) error {
		err := work(ctx, b)
		b.Log(text.String()) // A failed run prints it too.
		return err
	})
	if err == nil { // A work left behind may still write to text.
		_, _ = text.WriteTo(w.c.OutOrStdout())
	}
	return err
}

// whole runs every stage, on the run screen of their names.
func (w wizardScreen) whole() error {
	return w.running(wizardRun{o: w.o, sudo: &wizard.Sudo{}}, func(cfg tui.Config, work tui.Work) error {
		screen := tui.NewRunScreen(runTitle(w.c, w.o.check), cfg.Theme, tui.NewGroup("", w.plainStages()))
		return tui.Run(w.c.Context(), cfg, screen, work)
	})
}

// runOf is the run of what the lists chose. A stage in part leaves out ws's roles by --only, as they
// run on ws, and ws mac apply's steps by the run's filter. ws's run whole by a tick goes once more, as
// under --update: a good last run says nothing of a step that drifted after it.
func (w wizardScreen) runOf(choice tui.Choice, sudo *wizard.Sudo) wizardRun {
	o := w.o
	o.check = choice.Check
	name := wsRunOf(w.plainStages()).Name()
	o.update = o.update || slices.Contains(choice.Whole, name)
	return wizardRun{o: o, sudo: sudo, left: wizardLeft(choice, setupStepNames(w.p)), only: choice.Partial[name]}
}

// chosen runs what the lists chose, on a run of those stages.
func (w wizardScreen) chosen(choice tui.Choice, sudo *wizard.Sudo) error {
	r := w.runOf(choice, sudo)
	return w.running(r, func(cfg tui.Config, work tui.Work) error {
		screen := tui.NewQueueRun(runTitle(w.c, r.o.check), cfg.Theme, choice)
		return tui.Run(w.c.Context(), cfg, screen, work)
	})
}
