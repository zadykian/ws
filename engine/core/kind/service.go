package kind

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/resource/systemd"
)

// runs maps a service's run to systemd's: "" leaves the unit running or not.
var runs = map[string]systemd.Run{"": systemd.Leave, "running": systemd.Running, "stopped": systemd.Stopped,
	"triggered": systemd.Triggered}

// serviceSpec is a unit of the kind service: a systemd unit that ws owns, enabled, and running,
// stopped or triggered. Its files, File, are written in order before it, and those that go are
// removed after the rest. A change to any of them
// gives one restart, or one reload with Reload, owed to the next run where this one fails.
// InForce and Loads compare what is in force with what the files load (drift). A service whose
// files all go is gone: it stops and is disabled before they do.
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
	gone := s.gone()
	if gone && (s.Enabled || run != systemd.Leave && run != systemd.Stopped) {
		return nil, errors.New(`a service whose files are all absent takes no enabled, and run "stopped" or ""`)
	}
	var paths []string
	for i, f := range s.File {
		err := f.check(dir)
		if err == nil {
			err = f.checkService()
		}
		if err != nil {
			return nil, fmt.Errorf("file %d: %w", i+1, err)
		}
		if !f.Absent {
			paths = append(paths, f.Path)
		}
	}
	return func(ctx context.Context) ([]resource.Resource, error) {
		// A file goes after those the service writes, so that a run cut short, or a refused check,
		// never leaves a file that names one already gone.
		var all, removed, watch []resource.Resource
		for _, f := range s.File {
			b, err := f.resources(ctx, e, dir)
			if err != nil {
				return nil, err
			}
			if b.gone {
				removed = append(removed, b.all...)
			} else {
				all = addParts(all, b.all)
			}
			watch = append(watch, b.file)
		}
		all = append(all, removed...)
		if gone {
			// The stop and the disable come first, while the unit's file still names its ExecStop
			// and the links that [Install] made.
			unit := systemd.Unit{M: e.M, U: e.U, Name: s.Unit, Disabled: true, Run: systemd.Stopped}
			return append([]resource.Resource{unit}, all...), nil
		}
		if len(s.InForce) > 0 {
			d := drift{m: e.M, inForce: s.InForce, loads: s.Loads, files: paths}
			all, watch = append(all, d), append(watch, d)
		}
		return append(all, systemd.Unit{M: e.M, U: e.U, Name: s.Unit, Enabled: s.Enabled, Run: run,
			Reload: s.Reload, Watch: watch}), nil
	}, nil
}

// gone reports whether every file of the service goes, which takes the service off the machine.
func (s serviceSpec) gone() bool {
	return len(s.File) > 0 && !slices.ContainsFunc(s.File, func(f filePart) bool { return !f.Absent })
}

// addParts appends parts to all, less a directory that all makes already: two files of one new
// directory plan its creation both, and the second would find it made.
func addParts(all, parts []resource.Resource) []resource.Resource {
	for _, p := range parts {
		if d, ok := p.(file.Dir); ok && slices.ContainsFunc(all, func(r resource.Resource) bool {
			made, ok := r.(file.Dir)
			return ok && made.Path == d.Path
		}) {
			continue
		}
		all = append(all, p)
	}
	return all
}
