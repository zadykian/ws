//go:build !linux

package units

import (
	"context"
	"errors"

	"github.com/zadykian/ws/engine/core/effect"
)

// OSUnits is the effect.Units of a machine with no systemd: each call fails.
type OSUnits struct{}

var errNoSystemd = errors.New("systemd runs on Linux alone")

func (OSUnits) State(context.Context, string) (effect.UnitState, error) {
	return effect.UnitState{}, errNoSystemd
}

func (OSUnits) Enable(context.Context, ...string) error { return errNoSystemd }

func (OSUnits) Disable(context.Context, ...string) error { return errNoSystemd }

func (OSUnits) Reload(context.Context, string) error { return errNoSystemd }

func (OSUnits) Start(context.Context, string) error { return errNoSystemd }

func (OSUnits) Restart(context.Context, string) error { return errNoSystemd }

func (OSUnits) Stop(context.Context, string) error { return errNoSystemd }

func (OSUnits) ResetFailed(context.Context, string) error { return errNoSystemd }

func (OSUnits) DaemonReload(context.Context) error { return errNoSystemd }
