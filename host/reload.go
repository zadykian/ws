package host

import (
	"context"

	"github.com/zadykian/ws/effect"
)

// DaemonReload makes systemd read its unit files again where any of the watched resources has a
// change to make, such as a unit file that changed. It comes after them and before the units. It
// skips where systemd is not running.
type DaemonReload struct {
	M     effect.Machine
	U     effect.Units
	Watch []Resource
}

type reload struct{ u effect.Units }

func (reload) String() string { return "reload systemd's unit files" }

// Forced makes a failed run still reload, as a handler does.
func (reload) Forced() bool { return true }

func (c reload) Apply(ctx context.Context) error { return c.u.DaemonReload(ctx) }

// Plan plans a reload where a watched resource has a change.
func (d DaemonReload) Plan(ctx context.Context) ([]Change, error) {
	if !Booted(d.M) {
		return nil, nil
	}
	if changed, err := anyChange(ctx, d.Watch); err != nil || !changed {
		return nil, err
	}
	return []Change{reload{d.U}}, nil
}
