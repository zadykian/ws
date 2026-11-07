package main

import (
	"bytes"
	"context"
	"fmt"
	"slices"
	"syscall"

	tea "charm.land/bubbletea/v2"
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

// draw opens the list of the stages over a scan and runs what it chooses, or goes straight to the
// run under --check. Every stage runs on ws, so the list is ws's alone.
func (s serverScreen) draw(check bool) error {
	if !opensList(s.c, s.p, false) {
		return s.running(serverPick{check: check}, nil)
	}
	ctx, title := s.c.Context(), s.c.CommandPath()
	queue := wizardQueue(s.p, s.stages(serverPick{}))
	choice, err := tui.PickStages(ctx, s.config(), title, queue, false, s.scan())
	if err != nil || len(choice.Only) == 0 {
		return err
	}
	name := wsRunOf(s.stages(serverPick{})).Name()
	r := serverPick{check: choice.Check, update: slices.Contains(choice.Whole, name), only: choice.Partial[name]}
	return s.running(r, &choice)
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

// scan is a check of every stage, beside the list. ws speaks for its roles once its values are
// read, as the remote script's report takes ws_user from them.
func (s serverScreen) scan() tui.Work {
	return func(ctx context.Context, b *tui.Bridge) error {
		on, talks := s.on(ctx, b)
		stages := serverWizardStages(on, b, talks, b, nil, &wizard.Hand{}, serverPick{check: true})
		run := wsRunOf(stages)
		watch := afterStage{b, run.WaitsOn(), func() {
			states, err := run.Remote.State(ctx, s.p.version)
			reportRoles(run.Name(), states, err, b.Report)
		}}
		o := step.Options{Observer: watch, Check: true, Stop: b.Stop(ctx)}
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
