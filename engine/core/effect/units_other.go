//go:build !linux

package effect

import (
	"context"
	"errors"
)

// OSUnits is the Units of a machine with no systemd: each call fails.
type OSUnits struct{}

var errNoSystemd = errors.New("systemd runs on Linux alone")

func (OSUnits) State(context.Context, string) (UnitState, error) { return UnitState{}, errNoSystemd }

func (OSUnits) Enable(context.Context, ...string) error { return errNoSystemd }

func (OSUnits) Disable(context.Context, ...string) error { return errNoSystemd }

func (OSUnits) Reload(context.Context, string) error { return errNoSystemd }

func (OSUnits) Start(context.Context, string) error { return errNoSystemd }

func (OSUnits) Restart(context.Context, string) error { return errNoSystemd }

func (OSUnits) Stop(context.Context, string) error { return errNoSystemd }

func (OSUnits) ResetFailed(context.Context, string) error { return errNoSystemd }

func (OSUnits) DaemonReload(context.Context) error { return errNoSystemd }
