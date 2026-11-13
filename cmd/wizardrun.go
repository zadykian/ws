package cmd

import (
	"context"
	"fmt"
	"github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/client/macos/stages/host"
	"github.com/zadykian/ws/roles/server/ubuntu/stages"
	"os/signal"
	"slices"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
)

// wizardRun is a wizard's run at a terminal, over its side: the Mac's or ws's.
type wizardRun struct {
	c    *cobra.Command
	p    cli.Env
	side wizardSide
	o    wizardOptions
}

// start runs the wizard on lines, or on its lists at a terminal. The steps that before build run
// first, on lines, with the run's signals and terminal. A step that replaces ws, such as the update,
// must run before the lock, and before the screen leaves the terminal raw.
func (w wizardRun) start(before ...func(effect.Out) step.Step) error {
	// The run's context ends with a signal, and its lock goes with it.
	ctx, stop := signal.NotifyContext(w.c.Context(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	term := effect.NewTerminal(w.c.InOrStdin(), w.c.ErrOrStderr())
	out := progOut{term, w.c.CommandPath(), map[string]bool{}}
	if len(before) > 0 {
		steps := make([]step.Step, len(before))
		for i, build := range before {
			steps[i] = build(out)
		}
		if _, err := step.RunWith(ctx, steps, step.Options{Observer: out, Check: w.o.check}); err != nil {
			return unwrapStep(err)
		}
	}
	if engine.DrawsScreen(cli.CallOf(w.c, w.p)) {
		stop() // The screen takes the signals, and asks before a signal ends a run.
		return w.draw()
	}
	return w.lines(ctx, term, out)
}

// lines is the plain run: the stages under the lock, then what stays by hand.
func (w wizardRun) lines(ctx context.Context, term *effect.Terminal, out progOut) error {
	hand := &wizard.Hand{}
	closing := func(ctx context.Context) {
		hand.Close(ctx, w.c.OutOrStdout())
		term.Flush()
	}
	talks, _ := w.p.Machine.(effect.Foreground)
	stages := w.side.stages(wizardEffects{p: w.p, prog: w.c.CommandPath(), talks: talks, ask: signalPrompt{term},
		hidden: term, out: out, text: w.c.OutOrStdout(), obs: out, hand: hand, sudo: &host.Sudo{}}, w.o)
	_, err := wizard.Run(ctx, w.p.Machine, step.Options{Observer: out, Check: w.o.check}, stages, w.p.LockDir(),
		w.side.pid, closing)
	// The runner names the stage before the stage's own message.
	return unwrapStep(err)
}

// draw opens the lists over a scan and runs what they choose, or goes straight to the run under
// --check, as ws client macos apply does.
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
	sudo := &host.Sudo{}
	stages := w.plainStages()
	var run string // ws's run, found once a round; a side with none has no name to choose
	if r := wsRunOf(stages); r != nil {
		run = r.Name()
	}
	return tui.Round{
		Stages: wizardQueue(stages),
		Scan:   w.scan(sudo),
		Read:   func(side string) (tui.StateReader, func(context.Context)) { return w.side.read(side, sudo) },
		Run:    func(_ context.Context, choice tui.QueueChoice) error { return w.chosen(choice, run, sudo) },
	}
}

// config is what the wizard's screens draw with. A signal ends a plain wizard through its context,
// with status 1, so here too a signal is an error of the run, not engine.SignalExit.
func (w wizardRun) config() tui.Config {
	cfg := engine.ScreenConfig(cli.CallOf(w.c, w.p))
	cfg.Cut = func(s syscall.Signal) error { return fmt.Errorf("stopped by %v", s) }
	return cfg
}

// plainStages returns the stages for their names and machines alone: none of them runs.
func (w wizardRun) plainStages() []step.Step {
	return w.side.stages(wizardEffects{p: w.p, hand: &wizard.Hand{}, sudo: &host.Sudo{}}, w.o)
}

// effects returns the stages' effects on b, whose stdout waits for the screen's end.
func (w wizardRun) effects(b *tui.Bridge, sudo *host.Sudo) wizardEffects {
	p, on := w.p, b.Effects()
	p.Machine = on.M
	return wizardEffects{p: p, prog: w.c.CommandPath(), talks: on.Talks, ask: b, hidden: b, out: b, text: on.Text, obs: b,
		hand: &wizard.Hand{}, sudo: sudo, stop: b.Options(step.Options{}).Stop}
}

// scan is a check of every stage, beside the lists. ws speaks for its roles once the stage that
// installs it has read it, as ws's stages behind that need a ws at this version.
func (w wizardRun) scan(sudo *host.Sudo) tui.Work {
	check := w.o
	check.check = true
	return func(ctx context.Context, b *tui.Bridge) error {
		e := w.effects(b, sudo)
		stages := w.side.stages(e, check)
		o := b.Options(step.Options{Check: true})
		if run := wsRunOf(stages); run != nil {
			o.Observer = afterStage{b, run.WaitsOn(), func() {
				states, err := run.Ws.State(ctx, w.p.Version)
				reportRoles(run.Name(), states, err, b.Report)
			}}
		}
		_, err := wizard.Run(ctx, e.p.Machine, o, stages, w.p.LockDir(), w.side.pid, nil)
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

// wsRunOf returns the stage of ws's run, or nil where the list holds none.
func wsRunOf(list []step.Step) *stages.WsRun {
	for _, s := range list {
		if run, ok := s.(*stages.WsRun); ok {
			return run
		}
	}
	return nil
}

// work is a run of the stages by o beside a screen. What stays by hand goes to the run's stdout, as in
// a plain run.
func (w wizardRun) work(o wizardOptions, sudo *host.Sudo) tui.Work {
	return func(ctx context.Context, b *tui.Bridge) error {
		e := w.effects(b, sudo)
		e.events = b
		closing := func(ctx context.Context) { e.hand.Close(ctx, e.text) }
		so := b.Options(step.Options{Check: o.check, LeftOut: o.left})
		_, err := wizard.Run(ctx, e.p.Machine, so, w.side.stages(e, o), w.p.LockDir(), w.side.pid, closing)
		return unwrapStep(err)
	}
}

// running gives draw a run's config and the work of o. The run's stdout prints past the screen,
// which it would paint over.
func (w wizardRun) running(o wizardOptions, sudo *host.Sudo, draw func(tui.Config, tui.Work) error) error {
	cfg, done := engine.RunConfig(cli.CallOf(w.c, w.p), w.config())
	defer done()
	return draw(cfg, w.work(o, sudo))
}

// whole runs every stage, on the run screen of their names.
func (w wizardRun) whole() error {
	return w.running(w.o, &host.Sudo{}, func(cfg tui.Config, work tui.Work) error {
		screen := tui.NewRunScreen(engine.RunTitle(cli.CallOf(w.c, w.p), w.o.check), cfg.Theme, tui.NewGroup("", w.plainStages()))
		return tui.Run(w.c.Context(), cfg, screen, work)
	})
}

// optionsOf is the options of a run of what the lists chose, run being ws's run's name. A stage in part leaves out ws's roles by
// --only, as they run on ws, and ws client macos apply's steps by the run's filter. ws's run whole
// by a tick goes once more, as under --update: a good last run says nothing of a step that drifted
// after it.
func (w wizardRun) optionsOf(choice tui.QueueChoice, run string) wizardOptions {
	o := w.o
	o.check = choice.Next == tui.NextCheck
	o.update = o.update || slices.Contains(choice.Whole, run)
	o.left, o.only = wizardLeft(choice), choice.Partial[run]
	return o
}

// chosen runs what the lists chose, on a run of those stages.
func (w wizardRun) chosen(choice tui.QueueChoice, run string, sudo *host.Sudo) error {
	o := w.optionsOf(choice, run)
	return w.running(o, sudo, func(cfg tui.Config, work tui.Work) error {
		screen := tui.NewQueueRun(engine.RunTitle(cli.CallOf(w.c, w.p), o.check), cfg.Theme, choice)
		return tui.Run(w.c.Context(), cfg, screen, work)
	})
}
