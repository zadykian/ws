package cmd

import (
	"context"
	"errors"
	"io"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/stages/tools"
)

// wizardSetup returns the stage that runs the steps of ws client macos apply in the wizard's
// process. Until mac.conf has WS_PUBLIC_KEY, the Tunnel step prints the Mac's key and the commands
// that make it ws's peer at every run. The Tunnel stage takes that over, so the text goes nowhere
// then. file is the mac.conf to read, which is Answers' would-be file under --check. obs takes the
// steps' events, which name the stage: each of a stage's runs reports them again, a check before it
// applies. It stops as the run does, and leaves out the steps that left names.
func wizardSetup(e wizardEffects, left func(parent, name string) bool, file string) *tools.Setup {
	p := e.p
	s := &tools.Setup{Conf: file, M: p.machine}
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
		return macOf(p, c, p.machine, e.out, w).Run(ctx, macos.Options{Options: step.Options{
			Observer: e.obs, Check: check, Parent: s.Name(), Stop: e.stop, LeftOut: left,
		}})
	}
	return s
}
