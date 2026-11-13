package wizard

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"github.com/zadykian/ws/engine/core/remote"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/macos/conf"
)

// WsReach is ws's way to itself, for ws server ubuntu wizard: the scripts run as the user that runs
// ws, root there, and host_vars names the target and WS_USER.
type WsReach struct {
	M effect.Machine
	// Want reads host_vars' target and ws_user.
	Want func() (conf.Conf, error)
	// Here runs claude's login at the terminal.
	Here effect.Foreground
	Out  effect.Out
}

func (r WsReach) shell() remote.Shell {
	return remote.LocalShell{M: r.M, User: func() string { c, _ := r.Want(); return c.User }}
}

func (r WsReach) want() (conf.Conf, error) { return r.Want() }

func (r WsReach) user() (string, error) {
	c, err := r.Want()
	return cmp.Or(c.User, defaultUser()), err
}

func (WsReach) rerun() string { return "run sudo ws server ubuntu apply" }

// claude runs the login as WS_USER: runuser gives WS_USER's login shell in place of ssh's.
func (r WsReach) claude(ctx context.Context) error {
	user, err := r.user()
	if err != nil {
		return fmt.Errorf("%s: %w", claudeStage, err)
	}
	r.Out.Say("%s: claude auth login runs as WS_USER: take its steps at this terminal, "+
		"and open the address it shows in a browser", claudeStage)
	if err := r.Here.PassForeground(ctx, false, "runuser", "-l", user, "-c", "claude auth login"); err != nil {
		return errors.New(claudeStage + ": WS_USER's claude still has no login, as above; run the wizard again")
	}
	return nil
}
