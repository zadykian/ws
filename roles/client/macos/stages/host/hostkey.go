// Package host holds the wizard stages that reach ws from the Mac.
// They are the host key, the login as root and the sudo password.
// The Mac's reach and ways to ws serve them.
package host

import (
	"context"
	"errors"
	"github.com/zadykian/ws/engine/core/remote"
	guided "github.com/zadykian/ws/engine/core/wizard"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
)

// Key is the stage of ws's host key: the ed25519 key that ws offers, matched against ws's
// console. It goes in known_hosts under the name ws, which config.d/ws's HostKeyAlias gives each
// way to ws.
type Key struct {
	// Conf is mac.conf's path, Lock the run's lock, whose host.pub holds the key to fingerprint.
	Conf, Lock string
	// Tty tells whether a terminal is there to ask. Check is set under --check, which asks nothing.
	Tty, Check bool
	// Env looks up a variable of the environment.
	Env func(string) (string, bool)
	SSH remote.SSH
	M   effect.Machine
	P   effect.Prompt
	Out effect.Out
}

// Name is the stage's name.
func (*Key) Name() string { return stage.Host }

// Side is the Mac: known_hosts holds ws's host key.
func (*Key) Side() guided.Side { return stage.Mac }

// WaitsOn is ws client macos apply, which writes config.d/ws.
func (*Key) WaitsOn() string { return stage.Setup }

// hostAdd is the change that adds ws's key to known_hosts.
type hostAdd struct{ server, file string }

func (c hostAdd) String() string {
	return stage.Host + ": would add " + c.server + "'s key to " + c.file
}

// conf reads mac.conf.
func (h *Key) conf() (conf.Conf, error) { return conf.Read(h.M, h.Conf) }

// known tells whether known_hosts holds a key under ws. It reads no key of ws's, as ws's sshd counts
// each connection that doesn't log in against the Mac's address (PerSourcePenalties).
func (h *Key) known(ctx context.Context, file string) bool {
	return remote.KnownHost(ctx, h.M, file, h.SSH.Host)
}

// noWay is the stop where no way leads to ws, naming what ws's target needs.
func noWay(server string, c conf.Conf) error {
	how := "set those in server.conf"
	switch c.Target {
	case "lan":
		how += " as roles/server/ubuntu/network/router.md has them"
	case "mac-vm":
		how = "start the VM, or set WS_LAN to the Mac's network for it, as roles/client/macos/conf/README.md says"
	default:
		how += ", as roles/client/macos/conf/README.md says"
	}
	return errors.New(stage.Host + ": ssh finds no direct way to " + server + " from this network: no cable, no address in " +
		"WS_LAN and no WS_REMOTE_PORT; " + how)
}

// Plan finds whether known_hosts lacks ws's key. Under --check it also shows what a run would do.
func (h *Key) Plan(ctx context.Context) (step.Plan, error) {
	c, err := h.conf()
	if err != nil {
		return step.Plan{}, err
	}
	way, err := h.SSH.Way(ctx)
	if err != nil {
		return step.Plan{}, noWay(h.SSH.Host, c)
	}
	if h.known(ctx, way.KnownHosts) {
		return step.Plan{}, nil
	}
	if h.Check {
		// Under --check the plan shows what a run would ask, and the line it would add.
		if err := h.add(ctx); err != nil {
			return step.Plan{}, err
		}
	}
	return step.Plan{Changes: []step.Change{hostAdd{h.SSH.Host, way.KnownHosts}}}, nil
}
