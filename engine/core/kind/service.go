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
	Unit    string     `toml:"unit"`
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
	var paths []string
	for i, f := range s.File {
		if err := f.check(dir); err != nil {
			return nil, fmt.Errorf("file %d: %w", i+1, err)
		}
		if !f.Absent {
			paths = append(paths, f.Path)
		}
	}
	return func(ctx context.Context) ([]resource.Resource, error) {
		var all, watch []resource.Resource
		for _, f := range s.File {
			b, err := f.resources(ctx, e, dir)
			if err != nil {
				return nil, err
			}
			all, watch = append(all, b.all...), append(watch, b.file)
		}
		if len(s.InForce) > 0 {
			d := drift{m: e.M, out: e.Out, inForce: s.InForce, loads: s.Loads, files: paths}
			all, watch = append(all, d), append(watch, d)
		}
		return append(all, systemd.Unit{M: e.M, U: e.U, Out: e.Out, Name: s.Unit, Enabled: s.Enabled, Run: run,
			Reload: s.Reload, Watch: watch}), nil
	}, nil
}
