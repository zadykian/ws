package host

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/effect"
)

// Run is what a Unit wants of a unit's activity.
type Run int

// The activities. Leave touches neither.
const (
	Leave Run = iota
	Running
	Stopped
)

// Unit is a systemd unit that is enabled or not, and running or not. Without systemd booted, as in
// CI's container, it plans nothing and says so.
type Unit struct {
	M    effect.Machine
	U    effect.Units
	Out  effect.Out
	Name string
	// Enabled makes the unit start at boot.
	Enabled bool
	Run     Run
	// Watch restarts a running unit where any of these resources has a change to make, as a
	// handler does. They come before the unit in the group.
	Watch []Resource
}

type unitCall struct {
	u    Unit
	verb string
	do   func(context.Context) error
	// forced marks a restart, which Group.Apply still applies after a failure.
	forced bool
}

func (c unitCall) Forced() bool { return c.forced }

func (c unitCall) String() string { return c.verb + " " + c.u.Name }

func (c unitCall) Apply(ctx context.Context) error { return c.do(ctx) }

// Plan reads the unit's state over D-Bus.
func (u Unit) Plan(ctx context.Context) ([]Change, error) {
	if !Booted(u.M) {
		u.Out.Say("%s: skipped, systemd is not running", u.Name)
		return nil, nil
	}
	s, err := u.U.State(ctx, u.Name)
	if err != nil {
		return nil, fmt.Errorf("unit %s: %w", u.Name, err)
	}
	var cs []Change
	call := func(verb string, do func(context.Context, string) error) {
		cs = append(cs, unitCall{u, verb, func(ctx context.Context) error { return do(ctx, u.Name) }, verb == "restart"})
	}
	if u.Enabled && !s.Enabled {
		cs = append(cs, unitCall{u, "enable", func(ctx context.Context) error { return u.U.Enable(ctx, u.Name) }, false})
	}
	switch {
	case u.Run == Running && !s.Active:
		if s.Failed {
			call("reset failed state of", u.U.ResetFailed)
		}
		call("start", u.U.Start)
	case u.Run == Running:
		changed, err := u.watched(ctx)
		if err != nil {
			return nil, err
		}
		if changed {
			call("restart", u.U.Restart)
		}
	case u.Run == Stopped && s.Active:
		call("stop", u.U.Stop)
	}
	return cs, nil
}

func (u Unit) watched(ctx context.Context) (bool, error) {
	for _, r := range u.Watch {
		cs, err := r.Plan(ctx)
		if err != nil || len(cs) > 0 {
			return len(cs) > 0, err
		}
	}
	return false, nil
}
