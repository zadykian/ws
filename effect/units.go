package effect

import "context"

// UnitState is what systemd reports of a unit.
type UnitState struct {
	Enabled bool // the unit's file is enabled, so that boot starts it
	Active  bool // the unit runs
	Failed  bool // the unit failed and was not reset
	// NeedsReload is set where the unit's file changed after systemd read it, as `systemctl show`
	// reports NeedDaemonReload.
	NeedsReload bool
}

// Units drives systemd over D-Bus, not through systemctl, whose output is text for people.
type Units interface {
	// State reads the unit's state. A unit systemd does not know is neither enabled nor active.
	State(ctx context.Context, name string) (UnitState, error)
	// Enable enables the units' files and starts none.
	Enable(ctx context.Context, names ...string) error
	// Disable disables the units' files and stops none.
	Disable(ctx context.Context, names ...string) error
	// Start, Restart, Reload and Stop wait for systemd's job, and fail where it does not end done.
	Start(ctx context.Context, name string) error
	Restart(ctx context.Context, name string) error
	Reload(ctx context.Context, name string) error
	Stop(ctx context.Context, name string) error
	ResetFailed(ctx context.Context, name string) error
	// DaemonReload makes systemd read the unit files again.
	DaemonReload(ctx context.Context) error
}
