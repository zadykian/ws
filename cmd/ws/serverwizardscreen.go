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

// drawServerWizard runs ws server wizard's stages on the run screen of their names, as ws mac wizard
// does under --check. What stays by hand prints past the screen, which would paint over it.
func drawServerWizard(c *cobra.Command, p process, check bool, tmp string) error {
	cfg := screenConfig(c, p)
	// A signal ends a plain run through its context, with status 1, so here too it is the run's error.
	cfg.Cut = func(s syscall.Signal) error { return fmt.Errorf("stopped by %v", s) }
	cfg, done := runConfig(c, p, cfg)
	defer done()
	var text bytes.Buffer
	names := tui.NewGroup("", serverWizardStages(p, nil, nil, nil, nil, &wizard.Hand{}, serverPick{check: check}))
	err := tui.Run(c.Context(), cfg, tui.NewRunScreen(runTitle(c, check), cfg.Theme, names),
		func(ctx context.Context, b *tui.Bridge) error {
			on := p
			on.machine = screenMachine(ctx, b, p.machine)
			talks, _ := on.machine.(effect.Foreground)
			if m, ok := p.machine.(effect.OSMachine); ok {
				talks = b.Foreground(m)
			}
			hand := &wizard.Hand{}
			stages := serverWizardStages(on, b, talks, b, b, hand, serverPick{check: check})
			closing := func(ctx context.Context) { hand.Close(ctx, &text) }
			o := step.Options{Observer: b, Check: check, Stop: b.Stop(ctx)}
			_, err := wizard.Run(ctx, on.machine, o, stages, tmp, 0, closing)
			b.Log(text.String()) // A failed run prints it too.
			return unwrapStep(err)
		})
	if err == nil { // A work left behind may still write to text.
		_, _ = text.WriteTo(c.OutOrStdout())
	}
	return err
}
