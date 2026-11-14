package cli

import (
	"context"
	"errors"
	"io"
	"path"
	"strings"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/stages/tools"
)

// wizardSetup returns the stage that runs the steps of ws client macos apply in the wizard's
// process. Until the server's file has WS_PUBLIC_KEY, the Tunnel step prints the Mac's key and the commands
// that make it ws's peer at every run. The Tunnel stage takes that over, so the text goes nowhere
// then. mac is the file to read, and name the server whose file lies beside it: both are Answers'
// would-be files under --check. shown is the real mac.conf, whose directory a refusal of another
// server's file names in place of mac's, as the run's refusal does. The steps list the servers beside mac too, so that under --check
// they see the others that Answers copied there, as the run sees them. obs takes the steps'
// events, which name the stage: each of a stage's runs reports them again, a check before it
// applies. It stops as the run does, and leaves out the steps that left names.
func wizardSetup(e wizardEffects, left func(parent, name string) bool, mac, shown, name string) *tools.Setup {
	p := e.p
	s := &tools.Setup{Conf: mac, M: p.Machine}
	s.Run = func(ctx context.Context, check bool) (bool, error) {
		c, err := conf.Load(p.Machine, mac, conf.ServerPath(mac, name))
		if err != nil {
			return false, err
		}
		if err := refuseBadSibling(p.Machine, mac, name); err != nil {
			return false, atRealPath(err, mac, shown)
		}
		w := io.Discard
		if c.PublicKey != "" {
			w = e.text
		}
		m := macOf(p, name, c, p.Machine, e.out, w)
		m.MacConf = mac
		return m.Run(ctx, macos.Options{Options: step.Options{
			Observer: e.obs, Check: check, Parent: s.Name(), Stop: e.stop, LeftOut: left,
		}})
	}
	return s
}

// atRealPath words err as if it named the files beside shown, where it names those beside mac,
// which a --check run reads as copies in the lock.
func atRealPath(err error, mac, shown string) error {
	if mac == shown {
		return err
	}
	return errors.New(strings.ReplaceAll(err.Error(), path.Dir(mac)+"/", path.Dir(shown)+"/"))
}
