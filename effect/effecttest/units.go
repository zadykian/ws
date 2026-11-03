package effecttest

import (
	"context"
	"sync"

	"github.com/zadykian/ws/effect"
)

// Units keeps the state of units and changes it as systemd does. An unknown unit is inactive. A
// call with a name that has no unit type's suffix fails, as systemd's D-Bus API refuses it.
type Units struct {
	mu     sync.Mutex
	States map[string]effect.UnitState
	Calls  []string // each call, as "enable NAME", "start NAME", "daemon-reload" ...
}

// State returns the unit's state.
func (u *Units) State(_ context.Context, name string) (effect.UnitState, error) {
	if err := effect.CheckUnitName("state", name); err != nil {
		return effect.UnitState{}, err
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.States[name], nil
}

// Enable enables the units.
func (u *Units) Enable(_ context.Context, names ...string) error {
	for _, n := range names {
		if err := effect.CheckUnitName("enable", n); err != nil {
			return err
		}
	}
	for _, n := range names {
		u.change("enable", n, func(s *effect.UnitState) { s.Enabled = true })
	}
	return nil
}

// Disable disables the units.
func (u *Units) Disable(_ context.Context, names ...string) error {
	for _, n := range names {
		if err := effect.CheckUnitName("disable", n); err != nil {
			return err
		}
	}
	for _, n := range names {
		u.change("disable", n, func(s *effect.UnitState) { s.Enabled = false })
	}
	return nil
}

// Reload records the reload of a unit's configuration.
func (u *Units) Reload(_ context.Context, name string) error {
	if err := effect.CheckUnitName("reload", name); err != nil {
		return err
	}
	u.change("reload", name, func(*effect.UnitState) {})
	return nil
}

// Start runs the unit.
func (u *Units) Start(_ context.Context, name string) error {
	if err := effect.CheckUnitName("start", name); err != nil {
		return err
	}
	u.change("start", name, func(s *effect.UnitState) { s.Active, s.Failed = true, false })
	return nil
}

// Restart runs the unit, whatever it was.
func (u *Units) Restart(_ context.Context, name string) error {
	if err := effect.CheckUnitName("restart", name); err != nil {
		return err
	}
	u.change("restart", name, func(s *effect.UnitState) { s.Active, s.Failed = true, false })
	return nil
}

// Stop ends the unit.
func (u *Units) Stop(_ context.Context, name string) error {
	if err := effect.CheckUnitName("stop", name); err != nil {
		return err
	}
	u.change("stop", name, func(s *effect.UnitState) { s.Active = false })
	return nil
}

// ResetFailed clears the unit's failure.
func (u *Units) ResetFailed(_ context.Context, name string) error {
	if err := effect.CheckUnitName("reset-failed", name); err != nil {
		return err
	}
	u.change("reset-failed", name, func(s *effect.UnitState) { s.Failed = false })
	return nil
}

// DaemonReload records the reload, after which no unit needs one.
func (u *Units) DaemonReload(context.Context) error {
	u.change("daemon-reload", "", func(*effect.UnitState) {})
	u.mu.Lock()
	defer u.mu.Unlock()
	for n, s := range u.States {
		s.NeedsReload = false
		u.States[n] = s
	}
	return nil
}

func (u *Units) change(verb, name string, set func(*effect.UnitState)) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.States == nil {
		u.States = map[string]effect.UnitState{}
	}
	s := u.States[name]
	set(&s)
	if name != "" {
		u.States[name] = s
		verb += " " + name
	}
	u.Calls = append(u.Calls, verb)
}
