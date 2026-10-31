package wizard

import (
	"context"
	"errors"

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

// hostStage names the stage in its messages.
const hostStage = "Host key"

// HostKey is the stage of ws's host key: the ed25519 key that ws offers, matched against ws's
// console, in known_hosts under the name ws, which config.d/ws's HostKeyAlias gives each way to ws.
type HostKey struct {
	// Conf is mac.conf's path, Lock the run's lock, whose host.pub holds the key to fingerprint.
	Conf, Lock string
	// Tty tells whether a terminal is there to ask. Check is set under --check, which asks nothing.
	Tty, Check bool
	// Env looks up a variable of the environment.
	Env func(string) (string, bool)
	SSH SSH
	M   effect.Machine
	P   effect.Prompt
	Out effect.Out
}

// Name is the stage's name.
func (*HostKey) Name() string { return hostStage }

// WaitsOn is ws setup, which writes config.d/ws.
func (*HostKey) WaitsOn() string { return setupStage }

// hostAdd is the change that adds ws's key to known_hosts.
type hostAdd struct{ file string }

func (c hostAdd) String() string {
	return hostStage + ": would add ws's key to " + c.file
}

// conf reads mac.conf.
func (h *HostKey) conf() (conf.Conf, error) {
	text, err := h.M.ReadFile(h.Conf)
	if err != nil {
		return conf.Conf{}, err
	}
	c, errs := conf.Parse(string(text))
	if len(errs) > 0 {
		return c, errors.New(errs[0].In(h.Conf))
	}
	return c, nil
}

// known tells whether known_hosts holds a key under ws. It reads no key of ws's, as ws's sshd counts
// each connection that doesn't log in against the Mac's address (PerSourcePenalties).
func (h *HostKey) known(ctx context.Context, file string) bool {
	_, err := h.M.Run(ctx, "ssh-keygen", "-F", "ws", "-f", file)
	return err == nil
}

// noWay is the stop where no way leads to ws, naming what ws's target needs.
func noWay(c conf.Conf) error {
	how := "set those in mac.conf"
	switch c.Target {
	case "lan":
		how += " as docs/router.md has them"
	case "mac-vm":
		how = "start the VM, or set WS_LAN to the Mac's network for it, as docs/mac-targets.md says"
	default:
		how += ", as docs/mac-targets.md says"
	}
	return errors.New(hostStage + ": ssh finds no way to ws-ssh from this network: no cable, no address in " +
		"WS_LAN and no WS_REMOTE_PORT; " + how)
}

// Plan finds whether known_hosts lacks ws's key. Under --check it also shows what a run would do.
func (h *HostKey) Plan(ctx context.Context) (step.Plan, error) {
	c, err := h.conf()
	if err != nil {
		return step.Plan{}, err
	}
	way, err := h.SSH.Way(ctx)
	if err != nil {
		return step.Plan{}, noWay(c)
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
	return step.Plan{Changes: []step.Change{hostAdd{way.KnownHosts}}}, nil
}
