package main

import (
	"cmp"
	"context"
	"fmt"
	"io"
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

// config is what the wizard's screens draw with. A signal ends a plain wizard through its context,
// with status 1, so here too it is an error of the run, not signalExit.
func (w wizardScreen) config() tui.Config {
	cfg := screenConfig(w.c, w.p)
	cfg.Cut = func(s syscall.Signal) error { return fmt.Errorf("stopped by %v", s) }
	return cfg
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
