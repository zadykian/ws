package main

import (
	"bytes"
	"context"
	"fmt"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/effect"
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

// draw runs ws server wizard's stages on the run screen of their names, as ws mac wizard does under
// --check.
func (s serverScreen) draw(check bool) error {
	return s.running(serverPick{check: check})
}

// config is what the screens draw with. A signal ends a plain run through its context, with status
// 1, so here too it is the run's error.
func (s serverScreen) config() tui.Config {
	cfg := screenConfig(s.c, s.p)
	cfg.Cut = func(sig syscall.Signal) error { return fmt.Errorf("stopped by %v", sig) }
	return cfg
}

// stages returns the stages for their names and machines alone: none of them runs.
func (s serverScreen) stages(r serverPick) []step.Step {
	return serverWizardStages(s.p, nil, nil, nil, nil, &wizard.Hand{}, r)
}

// on returns the process and claude's terminal as a run beside the screen takes them.
func (s serverScreen) on(ctx context.Context, b *tui.Bridge) (process, effect.Foreground) {
	on := s.p
	on.machine = screenMachine(ctx, b, s.p.machine)
	talks, _ := on.machine.(effect.Foreground)
	if m, ok := s.p.machine.(effect.OSMachine); ok {
		talks = b.Foreground(m)
	}
	return on, talks
}

// running runs r on the run screen of the stages' names. What stays by hand prints past the screen,
// which would paint over it.
func (s serverScreen) running(r serverPick) error {
	cfg, done := runConfig(s.c, s.p, s.config())
	defer done()
	screen := tui.NewRunScreen(runTitle(s.c, r.check), cfg.Theme, tui.NewGroup("", s.stages(r)))
	o := step.Options{Check: r.check}
	var text bytes.Buffer
	err := tui.Run(s.c.Context(), cfg, screen, func(ctx context.Context, b *tui.Bridge) error {
		on, talks := s.on(ctx, b)
		hand := &wizard.Hand{}
		stages := serverWizardStages(on, b, talks, b, b, hand, r)
		closing := func(ctx context.Context) { hand.Close(ctx, &text) }
		o.Observer, o.Stop = b, b.Stop(ctx)
		_, err := wizard.Run(ctx, on.machine, o, stages, s.tmp, 0, closing)
		b.Log(text.String()) // A failed run prints it too.
		return unwrapStep(err)
	})
	if err == nil { // A work left behind may still write to text.
		_, _ = text.WriteTo(s.c.OutOrStdout())
	}
	return err
}
