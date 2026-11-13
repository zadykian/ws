package engine

import (
	"context"
	"fmt"
	"os/signal"
	"slices"
	"syscall"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/engine/tui"
)

// wizardRun is a wizard's run at a terminal, over its side: the Mac's or ws's.
type wizardRun struct {
	ctx context.Context
	c   Call
	t   Target
	o   WizardOptions
}

// Wizard runs t's wizard on lines, or on its lists at a terminal, as ws client macos wizard does.
func Wizard(ctx context.Context, c Call, t Target, o WizardOptions) error {
	return wizardRun{ctx, c, t, o}.start()
}

// start runs the wizard on lines, or on its lists at a terminal. The steps that before build run
// first, on lines, with the run's signals and terminal. A step that replaces ws, such as the update,
// must run before the lock, and before the screen leaves the terminal raw.
func (w wizardRun) start() error {
	// The run's context ends with a signal, and its lock goes with it.
	ctx, stop := signal.NotifyContext(w.ctx, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	term := effect.NewTerminal(w.c.In, w.c.Err)
	out := Lines{Out: term, Prog: w.c.Path, Seen: map[string]bool{}, Change: w.t.Wizard.Change}
	if before := w.sideBefore(); len(before) > 0 {
		steps := make([]step.Step, len(before))
		for i, build := range before {
			steps[i] = build(out)
		}
		if _, err := step.RunWith(ctx, steps, step.Options{Observer: out, Check: w.o.Check}); err != nil {
			return Cause(err)
		}
	}
	if DrawsScreen(w.c) {
		stop() // The screen takes the signals, and asks before a signal ends a run.
		return w.draw()
	}
	return w.lines(ctx, term, out)
}

// lines is the plain run: the stages under the lock, then what stays by hand.
func (w wizardRun) lines(ctx context.Context, term *effect.Terminal, out Lines) error {
	hand := &wizard.Hand{}
	closing := func(ctx context.Context) {
		hand.Close(ctx, w.c.Out)
		term.Flush()
	}
	talks, _ := w.c.Machine.(effect.Foreground)
	stages := w.t.Wizard.Stages(WizardEffects{Machine: w.c.Machine, Prog: w.c.Path, Talks: talks,
		Ask: SignalPrompt{Prompt: term}, Hidden: term, Out: out, Text: w.c.Out, Obs: out, Hand: hand, Sudo: w.newSudo()}, w.o)
	_, err := wizard.Run(ctx, w.c.Machine, step.Options{Observer: out, Check: w.o.Check}, stages, w.c.LockDir,
		w.t.Wizard.Pid, closing)
	// The runner names the stage before the stage's own message.
	return Cause(err)
}

// draw opens the lists over a scan and runs what they choose, or goes straight to the run under
// --check, as ws client macos apply does.
func (w wizardRun) draw() error {
	if !DrawsScreen(w.c) || w.c.Check {
		return w.whole()
	}
	return tui.RunWizard(w.ctx, tui.Wizard{Title: w.c.Path, Config: w.config(), Round: w.round,
		Forms: func(ctx context.Context) ([]tui.Sheet, func(context.Context), error) {
			return w.t.Sheets(ctx, w.c)
		}})
}

// round is the works of a scan. A new one reads the forms' answers. The state screen's read of ws and
// the run take the scan's sudo, which holds what it asked.
func (w wizardRun) round() tui.Round {
	sudo := w.newSudo()
	stages := w.plainStages()
	var run string // ws's run, found once a round; a side with none has no name to choose
	if r, ok := w.t.Wizard.WsRun(stages); ok {
		run = r.Name
	}
	return tui.Round{
		Stages: WizardQueue(w.t.Wizard, stages),
		Scan:   w.scan(sudo),
		Read:   func(side string) (tui.StateReader, func(context.Context)) { return w.t.Wizard.Read(side, sudo) },
		Run:    func(_ context.Context, choice tui.QueueChoice) error { return w.chosen(choice, run, sudo) },
	}
}

// config is what the wizard's screens draw with. A signal ends a plain wizard through its context,
// with status 1, so here too a signal is an error of the run, not SignalExit.
func (w wizardRun) config() tui.Config {
	cfg := ScreenConfig(w.c)
	cfg.Cut = func(s syscall.Signal) error { return fmt.Errorf("stopped by %v", s) }
	return cfg
}

// plainStages returns the stages for their names and machines alone: none of them runs.
func (w wizardRun) plainStages() []step.Step {
	return w.t.Wizard.Stages(WizardEffects{Machine: w.c.Machine, Hand: &wizard.Hand{}, Sudo: w.newSudo()}, w.o)
}

// effects returns the stages' effects on b, whose stdout waits for the screen's end.
func (w wizardRun) effects(b *tui.Bridge, sudo any) WizardEffects {
	on := b.Effects()
	return WizardEffects{Machine: on.M, Prog: w.c.Path, Talks: on.Talks, Ask: b, Hidden: b, Out: b,
		Text: on.Text, Obs: b, Hand: &wizard.Hand{}, Sudo: sudo, Stop: b.Options(step.Options{}).Stop}
}

// scan is a check of every stage, beside the lists. ws speaks for its roles once the stage that
// installs it has read it, as ws's stages behind that need a ws at this version.
func (w wizardRun) scan(sudo any) tui.Work {
	check := w.o
	check.Check = true
	return func(ctx context.Context, b *tui.Bridge) error {
		e := w.effects(b, sudo)
		stages := w.t.Wizard.Stages(e, check)
		o := b.Options(step.Options{Check: true})
		if run, ok := w.t.Wizard.WsRun(stages); ok {
			o.Observer = afterStage{b, run.Await, func() {
				states, err := run.State(ctx)
				ReportRoles(run.Name, states, err, b.Report)
			}}
		}
		_, err := wizard.Run(ctx, e.Machine, o, stages, w.c.LockDir, w.t.Wizard.Pid, nil)
		return Cause(err)
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

// work is a run of the stages by o beside a screen. What stays by hand goes to the run's stdout, as in
// a plain run.
func (w wizardRun) work(o WizardOptions, sudo any) tui.Work {
	return func(ctx context.Context, b *tui.Bridge) error {
		e := w.effects(b, sudo)
		e.Events = b
		closing := func(ctx context.Context) { e.Hand.Close(ctx, e.Text) }
		so := b.Options(step.Options{Check: o.Check, LeftOut: o.Left})
		_, err := wizard.Run(ctx, e.Machine, so, w.t.Wizard.Stages(e, o), w.c.LockDir, w.t.Wizard.Pid, closing)
		return Cause(err)
	}
}

// running gives draw a run's config and the work of o. The run's stdout prints past the screen,
// which it would paint over.
func (w wizardRun) running(o WizardOptions, sudo any, draw func(tui.Config, tui.Work) error) error {
	cfg, done := RunConfig(w.c, w.config())
	defer done()
	return draw(cfg, w.work(o, sudo))
}

// whole runs every stage, on the run screen of their names.
func (w wizardRun) whole() error {
	return w.running(w.o, w.newSudo(), func(cfg tui.Config, work tui.Work) error {
		screen := tui.NewRunScreen(RunTitle(w.c, w.o.Check), cfg.Theme, tui.NewGroup("", w.plainStages()))
		return tui.Run(w.ctx, cfg, screen, work)
	})
}

// optionsOf is the options of a run of what the lists chose, run being ws's run's name. A stage in part leaves out ws's roles by
// --only, as they run on ws, and ws client macos apply's steps by the run's filter. ws's run whole
// by a tick goes once more, as under --update: a good last run says nothing of a step that drifted
// after it.
func (w wizardRun) optionsOf(choice tui.QueueChoice, run string) WizardOptions {
	o := w.o
	o.Check = choice.Next == tui.NextCheck
	o.Update = o.Update || slices.Contains(choice.Whole, run)
	o.Left, o.Only = WizardLeft(choice), choice.Partial[run]
	return o
}

// chosen runs what the lists chose, on a run of those stages.
func (w wizardRun) chosen(choice tui.QueueChoice, run string, sudo any) error {
	o := w.optionsOf(choice, run)
	return w.running(o, sudo, func(cfg tui.Config, work tui.Work) error {
		screen := tui.NewQueueRun(RunTitle(w.c, o.Check), cfg.Theme, choice)
		return tui.Run(w.ctx, cfg, screen, work)
	})
}

// newSudo makes the sudo that a scan, the state read and the run share, or none.
func (w wizardRun) newSudo() any {
	if w.t.Wizard.NewSudo == nil {
		return nil
	}
	return w.t.Wizard.NewSudo()
}

// sideBefore builds the steps that run first.
func (w wizardRun) sideBefore() []func(effect.Out) step.Step {
	if w.t.Wizard.Before == nil {
		return nil
	}
	return w.t.Wizard.Before(w.o)
}
