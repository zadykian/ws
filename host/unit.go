package host

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// Run is what a Unit wants of a unit's activity.
type Run int

// The activities. Leave touches neither.
const (
	Leave Run = iota
	Running
	Stopped
	// Triggered restarts a unit on a watched change and never starts it, for a oneshot unit that a
	// timer or a path unit starts.
	Triggered
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
	// Disabled stops the unit from starting at boot, and leaves it running. It excludes Enabled.
	Disabled bool
	Run      Run
	// Reload makes a watched change reload a running unit, not restart it, as a reload handler does.
	Reload bool
	// Watch restarts a running unit where any of these resources has a change to make, as a
	// handler does. They come before the unit in the group.
	Watch []Resource
}

type unitCall struct {
	u    Unit
	verb string
	do   func(context.Context) error
	// forced marks a restart or a reload, which Group.Apply still applies after a failure.
	forced bool
}

func (c unitCall) Forced() bool { return c.forced }

func (c unitCall) String() string { return c.verb + " " + c.u.Name }

// Kind tells a change to what starts at boot, a link that systemd writes, from a change to what
// runs.
func (c unitCall) Kind() step.ChangeKind {
	if c.verb == "enable" || c.verb == "disable" {
		return step.Write
	}
	return step.Restart
}

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
	k := mark{u.M, u.Name}
	owed, err := k.owed()
	if err != nil {
		return nil, err
	}
	// call plans the verb. A restart or reload, or a start that an owed one makes needless, removes
	// the mark of the watched change once it is done.
	call := func(verb string, do func(context.Context, string) error) {
		forced := verb == "restart" || verb == "reload"
		settle := forced || (verb == "start" && owed)
		cs = append(cs, unitCall{u, verb, func(ctx context.Context) error {
			if err := do(ctx, u.Name); err != nil {
				return err
			}
			if settle {
				return k.settle(ctx)
			}
			return nil
		}, forced})
	}
	if s.NeedsReload {
		// A unit file written by a run that failed before the reload: the next run finds it right.
		cs = append(cs, reload{u.U})
	}
	if u.Enabled && !s.Enabled {
		cs = append(cs, unitCall{u, "enable", func(ctx context.Context) error { return u.U.Enable(ctx, u.Name) }, false})
	}
	if u.Disabled && s.Enabled {
		cs = append(cs, unitCall{u, "disable", func(ctx context.Context) error { return u.U.Disable(ctx, u.Name) }, false})
	}
	switch {
	case u.Run == Running && !s.Active:
		if s.Failed {
			call("reset failed state of", u.U.ResetFailed)
		}
		call("start", u.U.Start)
	case u.Run == Running:
		changed, err := anyChange(ctx, u.Watch)
		if err != nil {
			return nil, err
		}
		if changed && !owed {
			cs = append(cs, owe{k})
		}
		switch {
		case (changed || owed) && u.Reload:
			call("reload", u.U.Reload)
		case changed || owed:
			call("restart", u.U.Restart)
		}
	case u.Run == Triggered:
		changed, err := anyChange(ctx, u.Watch)
		if err != nil {
			return nil, err
		}
		if changed && !owed {
			cs = append(cs, owe{k})
		}
		if changed || owed {
			call("restart", u.U.Restart)
		}
	case u.Run == Stopped && s.Active:
		call("stop", u.U.Stop)
	case u.Run == Stopped && s.Failed:
		call("reset failed state of", u.U.ResetFailed)
	}
	return cs, nil
}

// anyChange reports whether any of the resources has a change to make.
func anyChange(ctx context.Context, rs []Resource) (bool, error) {
	for _, r := range rs {
		if cs, err := r.Plan(ctx); err != nil || len(cs) > 0 {
			return len(cs) > 0, err
		}
	}
	return false, nil
}
