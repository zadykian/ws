package cli

import (
	"context"
	"io"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/stages/tools"
)

// wizardSetup returns the stage that runs the steps of ws client macos apply in the wizard's
// process. Until mac.conf has WS_PUBLIC_KEY, the Tunnel step prints the Mac's key and the commands
// that make it ws's peer at every run. The Tunnel stage takes that over, so the text goes nowhere
// then. mac and server are the files to read, which are Answers' would-be files under --check. obs takes the
// steps' events, which name the stage: each of a stage's runs reports them again, a check before it
// applies. It stops as the run does, and leaves out the steps that left names.
func wizardSetup(e wizardEffects, left func(parent, name string) bool, mac, server string) *tools.Setup {
	p := e.p
	s := &tools.Setup{Conf: mac, M: p.Machine}
	s.Run = func(ctx context.Context, check bool) (bool, error) {
		c, err := conf.Load(p.Machine, mac, server)
		if err != nil {
			return false, err
		}
		w := io.Discard
		if c.PublicKey != "" {
			w = e.text
		}
		return macOf(p, c, p.Machine, e.out, w).Run(ctx, macos.Options{Options: step.Options{
			Observer: e.obs, Check: check, Parent: s.Name(), Stop: e.stop, LeftOut: left,
		}})
	}
	return s
}
