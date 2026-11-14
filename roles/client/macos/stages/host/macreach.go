package host

import (
	"context"
	"errors"
	"github.com/zadykian/ws/engine/core/remote"
	"github.com/zadykian/ws/roles/server/ubuntu/session"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/macos/conf"
)

// MacReach is the Mac's way to ws: ssh at the direct way that mac.conf gives, root by Sudo.
type MacReach struct {
	SSH remote.SSH
	// Sudo is how a command reaches root. Nil runs it as the login, as the tests do.
	Sudo *Sudo
	// Conf is the server file's path, which the reach reads for the target, WS_USER, WS_SLOT and WS_PORT.
	Conf string
	Out  effect.Out
}

// Shell runs a command as root on ws over the Mac's ssh, through sudo where WS_USER isn't root.
func (r MacReach) Shell() remote.Shell { return remote.SSHShell{SSH: r.SSH, Root: r.Sudo} }

// Want reads the target, WS_USER, WS_SLOT and WS_PORT from the server's conf.
func (r MacReach) Want() (session.Want, error) {
	c, err := conf.Read(r.SSH.M, r.Conf)
	return session.Want{Target: c.Target, User: c.User, Address: c.TunnelSlot().Wg0(), Port: c.Port}, err
}

// User returns WS_USER from mac.conf.
func (r MacReach) User() (string, error) {
	c, err := r.Want()
	return c.User, err
}

// Rerun says how to run ws server ubuntu apply again from the Mac.
func (MacReach) Rerun() string { return "run the wizard with --update" }

// Claude runs the login in a session of ssh -t, through a login shell, whose PATH has ~/.local/bin,
// where claude is.
func (r MacReach) Claude(ctx context.Context) error {
	r.Out.Say("%s: claude auth login runs as WS_USER on %s: take its steps at this terminal, "+
		"and open the address it shows in the Mac's browser", session.ClaudeStage, r.SSH.Host)
	if err := r.SSH.Pass(ctx, "-t", r.SSH.Host, `exec bash -lc "claude auth login"`); err != nil {
		return errors.New(session.ClaudeStage + ": WS_USER's claude on " + r.SSH.Host + " still has no login, as above; run the wizard again")
	}
	return nil
}
