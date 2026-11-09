package wizard

import (
	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/conf"
)

// MacReach is the Mac's way to ws: ssh at the direct way that mac.conf gives, root by Sudo.
type MacReach struct {
	SSH SSH
	// Sudo is how a command reaches root. Nil runs it as the login, as a Remote's tests do.
	Sudo *Sudo
	Out  effect.Out
}

func (r MacReach) shell() Shell { return SSHShell{SSH: r.SSH, Sudo: r.Sudo} }

func (r MacReach) want() (conf.Conf, error) { return macConf(r.SSH.M, r.SSH.Conf) }

// WsReach is ws's way to itself, for ws server wizard: the scripts run as the user that runs ws,
// root there, and host_vars names the target and WS_USER.
type WsReach struct {
	M effect.Machine
	// Want reads host_vars' target and ws_user.
	Want func() (conf.Conf, error)
	// Default is the server's default ws_user, what a host_vars with no such line gives.
	Default string
	// Here runs claude's login at the terminal.
	Here effect.Foreground
	Out  effect.Out
}

func (r WsReach) shell() Shell {
	return LocalShell{M: r.M, User: func() string { c, _ := r.Want(); return c.User }}
}

func (r WsReach) want() (conf.Conf, error) { return r.Want() }
