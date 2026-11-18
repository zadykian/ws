package kind

import (
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/systemd"
)

// runs maps a service's run to systemd's: "" leaves the unit running or not.
var runs = map[string]systemd.Run{"": systemd.Leave, "running": systemd.Running, "stopped": systemd.Stopped,
	"triggered": systemd.Triggered}

// serviceSpec is a unit of the kind service: a systemd unit that ws owns, enabled, and running,
// stopped or triggered. Its files, File, are written in order before it. A change to any of them
// gives one restart, or one reload with Reload, owed to the next run where this one fails.
// InForce and Loads compare what is in force with what the files load (drift). Without booted
// systemd it plans no unit and says so.
type serviceSpec struct {
	base
	Unit string `toml:"unit"`
	// User makes Unit one of that account's own systemd, kept running, whose files are its own. Its
	// unit file is left to whatever installs it.
	User    string     `toml:"user"`
	Enabled bool       `toml:"enabled"`
	Run     string     `toml:"run"`
	Reload  bool       `toml:"reload"`
	InForce []string   `toml:"in_force"`
	Loads   []string   `toml:"loads"`
	File    []filePart `toml:"file"`
}

func (s serviceSpec) parts(e Env, dir fs.FS) (func(context.Context) ([]resource.Resource, error), error) {
	if err := effect.CheckUnitName("unit", s.Unit); err != nil {
		return nil, err
	}
	run, ok := runs[s.Run]
	if !ok {
		return nil, fmt.Errorf(`run %q is none of "running", "stopped", "triggered" and ""`, s.Run)
	}
	if (len(s.InForce) == 0) != (len(s.Loads) == 0) {
		return nil, errors.New("in_force goes with loads")
	}
	if s.User != "" {
		if err := s.checkUser(); err != nil {
			return nil, err
		}
	}
	var paths []string
	for i, f := range s.File {
		if err := f.check(dir); err != nil {
			return nil, fmt.Errorf("file %d: %w", i+1, err)
		}
		if !f.Absent {
			paths = append(paths, f.Path)
		}
	}
	said := make([]bool, len(s.File))
	return func(ctx context.Context) ([]resource.Resource, error) {
		var all, watch []resource.Resource
		for i, f := range s.File {
			b, err := f.resources(ctx, e, dir)
			if err != nil {
				return nil, err
			}
			if b.left {
				f.sayLeft(e, &said[i])
			}
			if b.file == nil {
				continue
			}
			all, watch = append(all, b.all...), append(watch, b.file)
		}
		if s.User != "" {
			owner, err := ownerOf(ctx, e, s.User)
			if err != nil {
				return nil, err
			}
			return append(all, systemd.UserUnit{M: e.M, Account: owner, Out: e.Out, Name: s.Unit, Watch: watch}), nil
		}
		if len(s.InForce) > 0 {
			d := drift{m: e.M, out: e.Out, inForce: s.InForce, loads: s.Loads, files: paths}
			all, watch = append(all, d), append(watch, d)
		}
		return append(all, systemd.Unit{M: e.M, U: e.U, Out: e.Out, Name: s.Unit, Enabled: s.Enabled, Run: run,
			Reload: s.Reload, Watch: watch}), nil
	}, nil
}

// checkUser refuses the fields that a unit of an account's own systemd cannot take: UserUnit keeps
// it running and restarts it, and no more. Each of its files is the account's.
func (s serviceSpec) checkUser() error {
	switch {
	case s.User == "root":
		return errors.New("user takes an account other than root, whose units are the system's")
	case s.Enabled || s.Reload || len(s.InForce) > 0:
		return errors.New("a unit of a user's own systemd takes no enabled, reload or in_force")
	case s.Run != "running":
		return errors.New(`a unit of a user's own systemd takes run = "running"`)
	}
	if err := checkUser("user", s.User); err != nil {
		return err
	}
	for i, f := range s.File {
		if f.Owner != s.User {
			return fmt.Errorf("file %d: owner %q is not the unit's user %q", i+1, f.Owner, s.User)
		}
	}
	return nil
}
