package wizard

import (
	"context"
	"errors"
	"github.com/zadykian/ws/engine/core/remote"

	"github.com/zadykian/ws/engine/core/effect"
)

// MacReach is the Mac's way to ws: ssh at the direct way that mac.conf gives, root by Sudo.
type MacReach struct {
	SSH remote.SSH
	// Sudo is how a command reaches root. Nil runs it as the login, as the tests do.
	Sudo *Sudo
	// Conf is mac.conf's path, which the reach reads for the target and WS_USER.
	Conf string
	Out  effect.Out
}

func (r MacReach) Shell() remote.Shell { return remote.SSHShell{SSH: r.SSH, Root: r.Sudo} }

func (r MacReach) Want() (Want, error) {
	c, err := macConf(r.SSH.M, r.Conf)
	return Want{Target: c.Target, User: c.User}, err
}

func (r MacReach) User() (string, error) {
	c, err := r.Want()
	return c.User, err
}

func (MacReach) Rerun() string { return "run the wizard with --update" }

// Claude runs the login in a session of ssh -t, through a login shell, whose PATH has ~/.local/bin,
// where claude is.
func (r MacReach) Claude(ctx context.Context) error {
	r.Out.Say("%s: claude auth login runs as WS_USER on ws: take its steps at this terminal, "+
		"and open the address it shows in the Mac's browser", ClaudeStage)
	if err := r.SSH.Pass(ctx, "-t", "ws", `exec bash -lc "claude auth login"`); err != nil {
		return errors.New(ClaudeStage + ": WS_USER's claude on ws still has no login, as above; run the wizard again")
	}
	return nil
}
