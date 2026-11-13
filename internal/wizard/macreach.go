package wizard

import (
	"context"
	"errors"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/macos/conf"
)

// MacReach is the Mac's way to ws: ssh at the direct way that mac.conf gives, root by Sudo.
type MacReach struct {
	SSH SSH
	// Sudo is how a command reaches root. Nil runs it as the login, as the tests do.
	Sudo *Sudo
	Out  effect.Out
}

func (r MacReach) shell() Shell { return SSHShell{SSH: r.SSH, Sudo: r.Sudo} }

func (r MacReach) want() (conf.Conf, error) { return macConf(r.SSH.M, r.SSH.Conf) }

func (r MacReach) user() (string, error) {
	c, err := r.want()
	return c.User, err
}

func (MacReach) rerun() string { return "run the wizard with --update" }

// claude runs the login in a session of ssh -t, through a login shell, whose PATH has ~/.local/bin,
// where claude is.
func (r MacReach) claude(ctx context.Context) error {
	r.Out.Say("%s: claude auth login runs as WS_USER on ws: take its steps at this terminal, "+
		"and open the address it shows in the Mac's browser", claudeStage)
	if err := r.SSH.Pass(ctx, "-t", "ws", `exec bash -lc "claude auth login"`); err != nil {
		return errors.New(claudeStage + ": WS_USER's claude on ws still has no login, as above; run the wizard again")
	}
	return nil
}
