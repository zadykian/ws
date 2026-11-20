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

// State fails, as no systemd runs here.
func (OSUnits) State(context.Context, string) (effect.UnitState, error) {
	return effect.UnitState{}, errNoSystemd
}

// Enable fails, as State does.
func (OSUnits) Enable(context.Context, ...string) error { return errNoSystemd }

// Disable fails, as State does.
func (OSUnits) Disable(context.Context, ...string) error { return errNoSystemd }

// Reload fails, as State does.
func (OSUnits) Reload(context.Context, string) error { return errNoSystemd }

// Start fails, as State does.
func (OSUnits) Start(context.Context, string) error { return errNoSystemd }

// Restart fails, as State does.
func (OSUnits) Restart(context.Context, string) error { return errNoSystemd }

// Stop fails, as State does.
func (OSUnits) Stop(context.Context, string) error { return errNoSystemd }

// ResetFailed fails, as State does.
func (OSUnits) ResetFailed(context.Context, string) error { return errNoSystemd }

// DaemonReload fails, as State does.
func (OSUnits) DaemonReload(context.Context) error { return errNoSystemd }
