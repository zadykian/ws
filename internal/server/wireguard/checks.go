package wireguard

import (
	"context"
	"errors"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/values"
)

// settingsCheck refuses the values the role cannot use, as a plan error.
type settingsCheck struct{ w values.Wireguard }

func (c settingsCheck) Plan(context.Context) ([]host.Change, error) { return nil, Settings(c.w) }

// networkdCheck refuses a machine where systemd-networkd doesn't run, as on Ubuntu Desktop, whose
// NetworkManager does its work instead: nothing would create wg0. It checks nothing without
// systemd booted.
type networkdCheck struct {
	m effect.Machine
	u effect.Units
}

func (c networkdCheck) Plan(ctx context.Context) ([]host.Change, error) {
	if !host.Booted(c.m) {
		return nil, nil
	}
	s, err := c.u.State(ctx, "systemd-networkd.service")
	if err != nil {
		return nil, err
	}
	if !s.Active {
		return nil, errors.New("systemd-networkd is not running, so nothing would create wg0: the role expects networkd, as on Ubuntu Server, not NetworkManager, as on Ubuntu Desktop")
	}
	return nil, nil
}
