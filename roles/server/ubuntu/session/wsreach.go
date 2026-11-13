package session

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"github.com/zadykian/ws/engine/core/remote"

	"github.com/zadykian/ws/engine/core/effect"

	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// ClaudeStage names the stage of WS_USER's claude login, which both reaches say in their messages.
const ClaudeStage = "claude on ws"

// WsReach is ws's way to itself, for ws server ubuntu wizard: the scripts run as the user that runs
// ws, root there, and host_vars names the target and WS_USER.
type WsReach struct {
	M effect.Machine
	// Read reads host_vars' target and ws_user.
	Read func() (Want, error)
	// Here runs claude's login at the terminal.
	Here effect.Foreground
	Out  effect.Out
}

// Shell runs a command here as root, which ws server ubuntu wizard has on ws itself.
func (r WsReach) Shell() remote.Shell {
	return remote.LocalShell{M: r.M, User: func() string { c, _ := r.Read(); return c.User }}
}

// Want reads the target and WS_USER through Read.
func (r WsReach) Want() (Want, error) { return r.Read() }

// User returns WS_USER, or the default where host_vars names none.
func (r WsReach) User() (string, error) {
	c, err := r.Read()
	return cmp.Or(c.User, values.Default().User), err
}

// Rerun says how to run ws server ubuntu apply again on ws.
func (WsReach) Rerun() string { return "run sudo ws server ubuntu apply" }

// Claude runs the login as WS_USER: runuser gives WS_USER's login shell in place of ssh's.
func (r WsReach) Claude(ctx context.Context) error {
	user, err := r.User()
	if err != nil {
		return fmt.Errorf("%s: %w", ClaudeStage, err)
	}
	r.Out.Say("%s: claude auth login runs as WS_USER: take its steps at this terminal, "+
		"and open the address it shows in a browser", ClaudeStage)
	if err := r.Here.PassForeground(ctx, false, "runuser", "-l", user, "-c", "claude auth login"); err != nil {
		return errors.New(ClaudeStage + ": WS_USER's claude still has no login, as above; run the wizard again")
	}
	return nil
}
