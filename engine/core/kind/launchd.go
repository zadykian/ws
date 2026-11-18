package kind

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/resource/launchd"
	"github.com/zadykian/ws/engine/core/step"
)

// launchDaemons is where launchd reads the plists of the daemons that run as root from boot.
const launchDaemons = "/Library/LaunchDaemons"

var (
	// labelPrefix starts a daemon's label, a reversed domain, and ends in a dot.
	labelPrefix = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.-]*\.$`)
	// daemonName ends a daemon's label: no dot, so that a longer prefix's label is not one of these.
	daemonName = regexp.MustCompile(`^[a-z0-9-]+$`)
)

// launchdSpec is a unit of the kind launchd: a family of LaunchDaemons, each labelled Prefix and its
// name, and each running Program as root from boot. A family is one unit, so that a unit's name
// holds no daemon's. The unit owns each plist of Prefix: one that the list lacks is removed.
type launchdSpec struct {
	base
	Prefix  string          `toml:"prefix"`
	Program string          `toml:"program"`
	Daemon  []launchdDaemon `toml:"daemon"`
}

// launchdDaemon is one daemon of a launchd unit.
type launchdDaemon struct {
	Name string `toml:"name"`
	// Arguments is the command line, the unit's program first.
	Arguments []string `toml:"arguments"`
	// Log takes the daemon's output and errors, where set.
	Log string `toml:"log"`
	// RestartOn and ReloadOn are files and directories that the daemon reads. A change to one of
	// RestartOn restarts it, as a change to the program does. A change to one of ReloadOn sends it
	// SIGHUP.
	RestartOn []string `toml:"restart_on"`
	ReloadOn  []string `toml:"reload_on"`
}

// cleanPath refuses a path that is not clean and absolute, field naming it in the error.
func cleanPath(field, p string) error {
	if !path.IsAbs(p) || path.Clean(p) != p || p == "/" {
		return fmt.Errorf("%s %q is no clean absolute path below /", field, p)
	}
	return nil
}

func (s launchdSpec) parts(e Env, _ fs.FS) (func(context.Context) ([]resource.Resource, error), error) {
	if !labelPrefix.MatchString(s.Prefix) {
		return nil, fmt.Errorf("prefix %q is no start of a label: a reversed domain that ends in a dot", s.Prefix)
	}
	if err := cleanPath("program", s.Program); err != nil {
		return nil, err
	}
	var names []string
	for _, d := range s.Daemon {
		if err := s.check(d, names); err != nil {
			return nil, fmt.Errorf("daemon %q: %w", d.Name, err)
		}
		names = append(names, d.Name)
	}
	return func(context.Context) ([]resource.Resource, error) {
		rs := make([]resource.Resource, 0, len(s.Daemon)+1)
		for _, d := range s.Daemon {
			rs = append(rs, daemonUnit{e: e, label: s.Prefix + d.Name, program: s.Program, d: d})
		}
		return append(rs, prunedDaemons{e: e, prefix: s.Prefix, keep: names}), nil
	}, nil
}

// check refuses a daemon whose fields launchd or the unit cannot take. names are the daemons
// before it.
func (s launchdSpec) check(d launchdDaemon, names []string) error {
	switch {
	case !daemonName.MatchString(d.Name):
		return errors.New("a name takes a-z, 0-9 and '-'")
	case slices.Contains(names, d.Name):
		return errors.New("the name is a daemon's before it")
	case len(d.Arguments) == 0 || d.Arguments[0] != s.Program:
		return fmt.Errorf("arguments start with the program, %s", s.Program)
	}
	if d.Log != "" {
		if err := cleanPath("log", d.Log); err != nil {
			return err
		}
	}
	for _, p := range slices.Concat(d.RestartOn, d.ReloadOn) {
		if err := cleanPath("a path to watch", p); err != nil {
			return err
		}
	}
	return nil
}

// launchdAction is one call of launchctl that a daemon's plan makes.
type launchdAction struct {
	what string
	run  func(ctx context.Context) error
}

func (a launchdAction) String() string { return a.what }

func (launchdAction) Kind() step.ChangeKind { return step.Restart }

func (a launchdAction) Apply(ctx context.Context) error { return a.run(ctx) }

// daemonUnit is one daemon's resource: its plist, launchd's state of it, and its stamp.
type daemonUnit struct {
	e       Env
	label   string
	program string
	d       launchdDaemon
}

func (u daemonUnit) plist() string { return path.Join(launchDaemons, u.label+".plist") }

func (u daemonUnit) ctl() launchd.Launchctl { return launchd.Launchctl{M: u.e.M} }

// Plan reads as the user alone, and plans the first that holds of these:
//   - a plist to write, then a bootout where launchd has the daemon, and a bootstrap;
//   - a bootstrap where launchd has it not;
//   - a bootout and a bootstrap where launchd runs other arguments, from an older plist;
//   - a kickstart where the program or a file of restart_on differs from the stamp, or no stamp
//     is kept;
//   - a SIGHUP where a file of reload_on differs.
//
// The stamp is kept after, where it differs. Each plan checks the program's trust first, so that no
// plist, which launchd runs at the next boot, is written for a program it cannot trust.
func (u daemonUnit) Plan(ctx context.Context) ([]resource.Change, error) {
	if err := u.trusted(true); err != nil {
		return nil, err
	}
	cs, err := file.File{M: u.e.M, Path: u.plist(), Content: plistText(u.label, u.d), Mode: 0o644}.Plan(ctx)
	if err != nil {
		return nil, err
	}
	loaded, err := u.ctl().Print(ctx, u.label)
	if err != nil {
		return nil, err
	}
	now, err := u.stamp()
	if err != nil {
		return nil, err
	}
	kept, err := u.e.M.ReadRootFile(stampPath(u.label))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	was := parseStamp(kept)
	switch {
	case len(cs) > 0:
		if loaded.Found {
			cs = append(cs, u.bootout())
		}
		cs = append(cs, u.bootstrap())
	case !loaded.Found:
		cs = append(cs, u.bootstrap())
	case loaded.Arguments != nil && !slices.Equal(loaded.Arguments, u.d.Arguments):
		cs = append(cs, u.bootout(), u.bootstrap())
	case kept == nil || differs(was, now, slices.Concat([]string{u.program}, u.d.RestartOn)):
		cs = append(cs, u.kickstart())
	case differs(was, now, u.d.ReloadOn):
		cs = append(cs, u.reload())
	}
	if now.text() != string(kept) {
		cs = append(cs, keepStamp{u})
	}
	return cs, nil
}

func (u daemonUnit) bootout() launchdAction {
	return launchdAction{"boot out " + u.label, func(ctx context.Context) error { return u.ctl().Bootout(ctx, u.label) }}
}

func (u daemonUnit) bootstrap() launchdAction {
	return launchdAction{"bootstrap " + u.label + " from " + u.plist(), func(ctx context.Context) error {
		if err := u.trusted(false); err != nil {
			return err
		}
		return u.ctl().Bootstrap(ctx, u.plist())
	}}
}

func (u daemonUnit) kickstart() launchdAction {
	return launchdAction{"restart " + u.label + ": its program or a file of restart_on changed", func(ctx context.Context) error {
		if err := u.trusted(false); err != nil {
			return err
		}
		return u.ctl().Kickstart(ctx, u.label)
	}}
}

// reload sends SIGHUP, which a daemon that runs no process refuses: a kickstart starts it then,
// and it reads its files as it starts.
func (u daemonUnit) reload() launchdAction {
	return launchdAction{"send SIGHUP to " + u.label + ": a file of reload_on changed", func(ctx context.Context) error {
		if err := u.trusted(false); err != nil {
			return err
		}
		if u.ctl().Signal(ctx, u.label, "SIGHUP") == nil {
			return nil
		}
		return u.ctl().Kickstart(ctx, u.label)
	}}
}

// trusted fails unless the program and each directory above it are root's and written by root
// alone. launchd runs the program as root, and another user who could write one could replace it.
// With missing, a path that is not there passes: a plan may read before an earlier unit installs
// the program, so it checks the directories that are there. A start checks each path again.
func (u daemonUnit) trusted(missing bool) error {
	for p := u.program; ; p = path.Dir(p) {
		a, err := u.e.M.RootAttrs(p)
		if missing && errors.Is(err, fs.ErrNotExist) {
			if p == "/" {
				return nil
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("%s, which launchd runs as root: %w", u.program, err)
		}
		if a.UID != 0 || a.Mode.Perm()&0o022 != 0 {
			return fmt.Errorf("%s, on the way to %s, which launchd runs as root, is not root's alone: "+
				"owner, group and mode %d:%d %o", p, u.program, a.UID, a.GID, a.Mode.Perm())
		}
		if p == "/" {
			return nil
		}
	}
}

// prunedDaemons removes each daemon of the prefix that the unit's list lacks: launchd's job, its
// plist and its stamp.
type prunedDaemons struct {
	e      Env
	prefix string
	keep   []string
}

// Plan lists the plists and the stamps, both readable by all.
func (p prunedDaemons) Plan(ctx context.Context) ([]resource.Change, error) {
	var cs []resource.Change
	for _, dir := range []struct{ path, suffix string }{{launchDaemons, ".plist"}, {stampDir, ""}} {
		entries, err := p.e.M.ReadDir(dir.path)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
		for _, entry := range entries {
			label, ok := strings.CutSuffix(entry, dir.suffix)
			name, ours := strings.CutPrefix(label, p.prefix)
			if !ok || !ours || !daemonName.MatchString(name) || slices.Contains(p.keep, name) {
				continue
			}
			if dir.path == launchDaemons {
				u := daemonUnit{e: p.e, label: label}
				loaded, err := u.ctl().Print(ctx, label)
				if err != nil {
					return nil, err
				}
				if loaded.Found {
					cs = append(cs, u.bootout())
				}
			}
			more, err := file.Absent{M: p.e.M, Path: path.Join(dir.path, entry)}.Plan(ctx)
			if err != nil {
				return nil, err
			}
			cs = append(cs, more...)
		}
	}
	return cs, nil
}
