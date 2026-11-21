//go:build !linux

package units

import (
	"context"
	"errors"

	"github.com/zadykian/ws/engine/core/effect"
)

// OSUnits is the effect.Units of a machine with no systemd: each call fails.
type OSUnits struct{}

var errNotLinux = errors.New("systemd runs on Linux alone")

// State fails, as no systemd runs here.
func (OSUnits) State(context.Context, string) (effect.UnitState, error) {
	return effect.UnitState{}, errNotLinux
}

// Enable fails, as State does.
func (OSUnits) Enable(context.Context, ...string) error { return errNotLinux }

// Disable fails, as State does.
func (OSUnits) Disable(context.Context, ...string) error { return errNotLinux }

// Reload fails, as State does.
func (OSUnits) Reload(context.Context, string) error { return errNotLinux }

// Start fails, as State does.
func (OSUnits) Start(context.Context, string) error { return errNotLinux }

// Restart fails, as State does.
func (OSUnits) Restart(context.Context, string) error { return errNotLinux }

// Stop fails, as State does.
func (OSUnits) Stop(context.Context, string) error { return errNotLinux }

// ResetFailed fails, as State does.
func (OSUnits) ResetFailed(context.Context, string) error { return errNotLinux }

// DaemonReload fails, as State does.
func (OSUnits) DaemonReload(context.Context) error { return errNotLinux }
