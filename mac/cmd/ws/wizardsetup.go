package main

import (
	"context"
	"errors"
	"io"
	"slices"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/wizard"
	"github.com/zadykian/ws/step"
)

// wizardSetup returns the stage that runs the steps of ws mac apply in the wizard's process. Until
// mac.conf has WS_PUBLIC_KEY, the Tunnel step prints the Mac's key and the commands that make it
// ws's peer at every run, which the Tunnel stage takes over: the text goes nowhere then. file is the
// mac.conf to read, which is Answers' would-be file under --check.
func wizardSetup(e wizardEnv, out effect.Out, file string) *wizard.Setup {
	return &wizard.Setup{M: e.m, Run: func(ctx context.Context, check bool) (bool, error) {
		text, err := e.m.ReadFile(file)
		if err != nil {
			return false, err
		}
		c, errs := conf.Parse(string(text))
		if len(errs) > 0 {
			return false, errors.New(errs[0].In(file))
		}
		w := io.Discard
		if c.PublicKey != "" && e.stdout != nil {
			w = e.stdout
		}
		home := e.getenv("HOME")
		steps := setupSteps(c, home, e.getenv("XDG_CONFIG_HOME"), e.m, out, w, check)
		if !check {
			// The tunnel's daemon and cld's restore tabs change ssh ws's ways, so ssh runs once more.
			steps = slices.Concat(steps, []step.Step{sshStep(c, home, e.m, out, check, true)})
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
