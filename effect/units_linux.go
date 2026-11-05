//go:build linux

package effect

import (
	"context"
	"fmt"
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
	if err := CheckUnitName("state", name); err != nil {
		return UnitState{}, err
	}
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

// Enable enables the units' files, and reloads systemd, as systemctl does: D-Bus alone leaves every
// unit reporting NeedDaemonReload, so a run would plan a reload that it cannot clear.
func (OSUnits) Enable(ctx context.Context, names ...string) error {
	for _, n := range names {
		if err := CheckUnitName("enable", n); err != nil {
			return err
		}
	}
	c, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return err
	}
	defer c.Close()
	if _, _, err = c.EnableUnitFilesContext(ctx, names, false, false); err != nil {
		return err
	}
	return c.ReloadContext(ctx)
}

// Disable disables the units' files, and reloads systemd, as Enable does.
func (OSUnits) Disable(ctx context.Context, names ...string) error {
	for _, n := range names {
		if err := CheckUnitName("disable", n); err != nil {
			return err
		}
	}
	c, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return err
	}
	defer c.Close()
	if _, err = c.DisableUnitFilesContext(ctx, names, false); err != nil {
		return err
	}
	return c.ReloadContext(ctx)
}

// Start starts the unit, replacing the jobs that conflict, and waits for the job.
func (OSUnits) Start(ctx context.Context, name string) error {
	if err := CheckUnitName("start", name); err != nil {
		return err
	}
	return job(ctx, name, func(c *dbus.Conn) startFunc { return c.StartUnitContext })
}

// Restart restarts the unit and waits for the job.
func (OSUnits) Restart(ctx context.Context, name string) error {
	if err := CheckUnitName("restart", name); err != nil {
		return err
	}
	return job(ctx, name, func(c *dbus.Conn) startFunc { return c.RestartUnitContext })
}

// Reload asks the unit to read its configuration again and waits for the job.
func (OSUnits) Reload(ctx context.Context, name string) error {
	if err := CheckUnitName("reload", name); err != nil {
		return err
	}
	return job(ctx, name, func(c *dbus.Conn) startFunc { return c.ReloadUnitContext })
}

// Stop stops the unit and waits for the job.
func (OSUnits) Stop(ctx context.Context, name string) error {
	if err := CheckUnitName("stop", name); err != nil {
		return err
	}
	return job(ctx, name, func(c *dbus.Conn) startFunc { return c.StopUnitContext })
}

type startFunc func(ctx context.Context, name, mode string, ch chan<- string) (int, error)

func job(ctx context.Context, name string, method func(*dbus.Conn) startFunc) error {
	c, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return err
	}
	defer c.Close()
	done := make(chan string, 1)
	if _, err := method(c)(ctx, name, "replace", done); err != nil {
		return err
	}
	select {
	case result := <-done:
		if result != "done" {
			return fmt.Errorf("systemd job for %s ended %s", name, result)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ResetFailed clears the unit's failed state.
func (OSUnits) ResetFailed(ctx context.Context, name string) error {
	if err := CheckUnitName("reset-failed", name); err != nil {
		return err
	}
	c, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return err
	}
	defer c.Close()
	return c.ResetFailedUnitContext(ctx, name)
}

// DaemonReload makes systemd read the unit files again.
func (OSUnits) DaemonReload(ctx context.Context) error {
	c, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return err
	}
	defer c.Close()
	return c.ReloadContext(ctx)
}
