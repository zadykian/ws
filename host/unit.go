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
}

type unitCall struct {
	u    Unit
	verb string
	do   func(context.Context) error
}

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
		cs = append(cs, unitCall{u, verb, func(ctx context.Context) error { return do(ctx, u.Name) }})
	}
	if u.Enabled && !s.Enabled {
		cs = append(cs, unitCall{u, "enable", func(ctx context.Context) error { return u.U.Enable(ctx, u.Name) }})
	}
	switch {
	case u.Run == Running && !s.Active:
		call("start", u.U.Start)
	case u.Run == Stopped && s.Active:
		call("stop", u.U.Stop)
	}
	return cs, nil
}
