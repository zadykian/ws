package main

import (
	"context"
	"errors"
	"io"
	"slices"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/step"
)

// wizardSetup returns the stage that runs the steps of ws mac apply in the wizard's process. Until
// mac.conf has WS_PUBLIC_KEY, the Tunnel step prints the Mac's key and the commands that make it
// ws's peer at every run, which the Tunnel stage takes over: the text goes nowhere then. file is the
// mac.conf to read, which is Answers' would-be file under --check.
func wizardSetup(p process, stdout io.Writer, out effect.Out, file string) *wizard.Setup {
	return &wizard.Setup{M: p.machine, Run: func(ctx context.Context, check bool) (bool, error) {
		text, err := p.machine.ReadFile(file)
		if err != nil {
			return false, err
		}
		c, errs := conf.Parse(string(text))
		if len(errs) > 0 {
			return false, errors.New(errs[0].In(file))
		}
		w := io.Discard
		if c.PublicKey != "" {
			w = stdout
		}
		home := p.getenv("HOME")
		steps := setupSteps(c, home, p.getenv("XDG_CONFIG_HOME"), p.machine, out, w, check)
		if !check {
			// cld's restore tabs change ssh's file, so ssh runs once more.
			steps = slices.Concat(steps, []step.Step{sshStep(c, home, p.machine, out, true)})
		}
		changed, err := step.Run(ctx, steps, out, check)
		for _, s := range steps {
			if f, ok := s.(interface{ Finish(context.Context) error }); ok && err == nil {
				err = f.Finish(ctx)
			}
		}
		return changed, err
	}}
}
