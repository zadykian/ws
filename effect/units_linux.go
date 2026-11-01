//go:build linux

package effect

import (
	"context"
	"path"
	"slices"

	"github.com/coreos/go-systemd/v22/dbus"
)

// OSUnits is the Units of the machine that ws runs on, over systemd's D-Bus on the system bus.
type OSUnits struct{}

// enabledFileStates are the states of a unit file that boot starts it in, or needs no enabling in.
var enabledFileStates = []string{"enabled", "enabled-runtime", "static", "alias", "generated", "transient", "indirect"}

func unitState(files []dbus.UnitFile, units []dbus.UnitStatus, name string) UnitState {
	var s UnitState
	for _, f := range files {
		if path.Base(f.Path) == name {
			s.Enabled = slices.Contains(enabledFileStates, f.Type)
		}
	}
	for _, u := range units {
		if u.Name == name {
			s.Active, s.Failed = u.ActiveState == "active", u.ActiveState == "failed"
		}
	}
	return s
}

// State asks systemd for the unit's file and for its active state.
func (OSUnits) State(ctx context.Context, name string) (UnitState, error) {
	c, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return UnitState{}, err
	}
	defer c.Close()
	files, err := c.ListUnitFilesByPatternsContext(ctx, nil, []string{name})
	if err != nil {
		return UnitState{}, err
	}
	units, err := c.ListUnitsByNamesContext(ctx, []string{name})
	if err != nil {
		return UnitState{}, err
	}
	s := unitState(files, units, name)
	if len(units) == 0 {
		return s, nil
	}
	p, err := c.GetUnitPropertyContext(ctx, name, "NeedDaemonReload")
	if err != nil {
		return UnitState{}, err
	}
	s.NeedsReload, _ = p.Value.Value().(bool)
	return s, nil
}
