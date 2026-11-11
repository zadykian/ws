package main

import (
	"context"
	"errors"
	"io"
	"slices"

	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/step"
)

// wizardSetup returns the stage that runs the steps of ws mac apply in the wizard's process. Until
// mac.conf has WS_PUBLIC_KEY, the Tunnel step prints the Mac's key and the commands that make it
// ws's peer at every run. The Tunnel stage takes that over, so the text goes nowhere then. file is
// the mac.conf to read, which is Answers' would-be file under --check. obs takes the steps' events,
// which name the stage: each of a stage's runs reports them again, a check before it applies. It
// stops and leaves steps out as the run does.
func wizardSetup(e wizardEffects, file string) *wizard.Setup {
	p := e.p
	s := &wizard.Setup{Conf: file, M: p.machine}
	s.Run = func(ctx context.Context, check bool) (bool, error) {
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
			w = e.text
		}
		home := p.getenv("HOME")
		steps := setupSteps(c, home, p.getenv("XDG_CONFIG_HOME"), p.machine, e.out, w, check)
		if !check {
			// cld's restore tabs change ssh's file, so ssh runs once more.
			steps = slices.Concat(steps, []step.Step{sshStep(c, home, p.machine, e.out, true)})
		}
		return finishSetup(ctx, steps, step.Options{Observer: e.obs, Check: check, Parent: s.Name(), Stop: e.stop, LeftOut: e.left})
	}
	return s
}
