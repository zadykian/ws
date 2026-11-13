package systemd

import (
	"context"
	"errors"
	"github.com/zadykian/ws/engine/core/host"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// Linger enables lingering for the user, which starts the user's systemd at boot. loginctl starts
// it now too.
type Linger struct {
	M    effect.Machine
	User host.User
	Out  effect.Out
}

type enableLinger struct{ l Linger }

func (c enableLinger) String() string { return "enable lingering for " + c.l.User.Name }

func (enableLinger) Kind() step.ChangeKind { return step.Write }

func (c enableLinger) Apply(ctx context.Context) error {
	_, err := c.l.M.RunRoot(ctx, "loginctl", "enable-linger", c.l.User.Name)
	return err
}

// Plan looks for the file that loginctl makes.
func (l Linger) Plan(context.Context) ([]host.Change, error) {
	if !Booted(l.M) {
		l.Out.Say("lingering for %s: skipped, systemd is not running", l.User.Name)
		return nil, nil
	}
	_, err := l.M.Stat("/var/lib/systemd/linger/" + l.User.Name)
	if errors.Is(err, fs.ErrNotExist) {
		return []host.Change{enableLinger{l}}, nil
	}
	return nil, err
}
