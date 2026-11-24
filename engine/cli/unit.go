package cli

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/kind"
)

// InSync is the state that ws unit read prints for a unit with nothing to change: the version that
// mise's plan pins each unit to.
const InSync = "in-sync"

// UnitSide is what a side gives ws unit: where its specs sit, the values that fill them, and the
// units of its service manager.
type UnitSide struct {
	// Name is the side's name, which a unit's side fact names. Empty is a machine with one side.
	Name string
	// Dir is the root of the side's specs, DIR/ROLE/units.toml.tmpl, where WS_UNITS names none.
	Dir string
	// RoleFile is the file that a role holds where the side runs it, as mise.host.toml: the side's
	// specs are those of the roles that hold it. Empty is every role.
	RoleFile string
	// RunDir is the directory mise runs in, which holds the side's conf.d link, lock and tasks.
	// Empty is the parent of Dir. It tells the sides of one ws unit apart, so each names its own.
	RunDir string
	// Values loads the side's values.
	Values func(ctx context.Context, m effect.Machine) (any, error)
	// Units is the service manager's units.
	Units effect.Units
	// Root refuses read and write to any user but root, as the side's units write root's paths.
	Root bool
	// Platform refuses read and write on a machine whose system the side's units are not written
	// for, or returns nil. Nil is a side that runs on any.
	Platform func(m effect.Machine) error
	// System is the side's operating system, which decides how the kinds find a home and run a
	// command as a user.
	System kind.System
	// User returns the account that ws runs as, from the values, where it runs as a user: the
	// Mac's. A run state of that user's needs no sudo there. Nil is a side that runs as root.
	User func(data any) string
	// Hidden names the variables of the run's env that hold the side's secrets. A run state drops
	// them from each command but ws's, and a template's envvar reads them in the run of Task, where
	// it sets them (kind.Env.Hidden).
	Hidden []string
	// Task is the mise task whose run gives the hidden variables their values (Env.TaskEnv).
	Task string
}

// unitRun is one call of ws unit. It holds the units of the specs in run order, ws's lines, which go
// to stderr, the specs' directory and the directory mise runs in.
type unitRun struct {
	all  []kind.Unit
	term *effect.Terminal
	out  engine.Lines
	dir  string
	run  string
}

// runDir is the directory mise runs in for the side, whose mise/conf.d links the specs. A run
// elsewhere loads none of their lines, so a failure for a line names it.
func (r unitRun) runDir() string {
	return r.run
}

// sides are the sides that one ws unit serves.
type sides []UnitSide

// pick returns the side whose run directory is pwd. One side serves a run anywhere. With several,
// a pwd in none of their run directories is an error that names them.
func (ss sides) pick(pwd string) (UnitSide, error) {
	if len(ss) == 1 {
		return ss[0], nil
	}
	dirs := make([]string, 0, len(ss))
	for _, s := range ss {
		if sameDir(pwd, s.RunDir) {
			return s, nil
		}
		dirs = append(dirs, s.RunDir)
	}
	return UnitSide{}, fmt.Errorf("mise runs in %s, which is no side's run directory: run mise in %s", pwd,
		strings.Join(dirs, " or "))
}

// Unit returns ws unit: the units of ws's kinds, KIND/NAME, which a tool such as mise plans and
// applies through it. With several sides it serves the one whose run directory mise runs in.
func Unit(p Env, first UnitSide, more ...UnitSide) *cobra.Command {
	ss := append(sides{first}, more...)
	return Group("unit", "plan and apply the units of ws's kinds, KIND/NAME, for mise",
		unitList(p, ss), unitRead(p, ss), unitWrite(p, ss), unitVerify(p, ss))
}

// unitList returns ws unit list.
func unitList(p Env, ss sides) *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "print each unit, KIND/NAME, in the order the units run",
		Long: "The specs are each units.toml.tmpl one level below WS_UNITS, or below " + ss[0].Dir + " where it is " +
			"unset. --toml prints each unit as a line of mise's [bootstrap.packages].",
		Args: cobra.NoArgs,
	}
	asTOML := c.Flags().Bool("toml", false, "print the units as lines of mise's [bootstrap.packages]")
	c.RunE = func(c *cobra.Command, _ []string) error {
		s, err := ss.pick(p.Pwd)
		if err != nil {
			return err
		}
		r, err := loadUnits(c, p, s)
		defer r.term.Flush()
		if err != nil {
			return err
		}
		var sb strings.Builder
		for _, name := range kind.Names(r.all) {
			if *asTOML {
				name = strconv.Quote("ws:"+name) + " = " + strconv.Quote(InSync)
			}
			sb.WriteString(name + "\n")
		}
		_, err = fmt.Fprint(c.OutOrStdout(), sb.String())
		return err
	}
	return c
}

// loadUnits loads the side's values and every unit of its specs, which name this ws by self. A spec
// that fails to load fails the command with status 1, as the specs and the values are the
// machine's, not the command line's.
func loadUnits(c *cobra.Command, p Env, s UnitSide) (unitRun, error) {
	term := effect.NewTerminal(c.ErrOrStderr())
	r := unitRun{term: term, out: engine.Lines{Out: term, Prog: c.CommandPath(), Seen: map[string]bool{}}}
	data, err := s.Values(c.Context(), p.Machine)
	if err != nil {
		return r, err
	}
	dir := cmp.Or(p.Getenv("WS_UNITS"), s.Dir)
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(p.Pwd, dir)
	}
	r.dir = dir
	r.run = cmp.Or(s.RunDir, filepath.Dir(filepath.Clean(dir)))
	specs := SideRoles(os.DirFS(dir), s.RoleFile)
	// fs.Glob passes over a directory it cannot read, which would give no unit and no error.
	if _, err := fs.Stat(specs, "."); err != nil {
		return r, fmt.Errorf("the specs: %w", err)
	}
	self, err := p.Self()
	if err != nil {
		return r, fmt.Errorf("finding the file ws runs from: %w", err)
	}
	env := kind.Env{M: p.Machine, U: s.Units, Out: r.out, Data: data, System: s.System, Side: s.Name, Self: self,
		Hidden: s.Hidden, LookupEnv: p.TaskEnv(s.Task)}
	if s.User != nil {
		env.User = s.User(data)
	}
	r.all, err = kind.Load(env, specs)
	if err != nil {
		return r, fmt.Errorf("%s: %w", dir, err)
	}
	return r, nil
}

// closure returns the named units and those they wait on. A name that no spec holds is a usage
// error.
func (r unitRun) closure(names []string) ([]kind.Unit, error) {
	units, err := kind.Closure(r.all, names)
	var unknown *kind.UnknownUnit
	if errors.As(err, &unknown) {
		return nil, engine.UsageError{Err: fmt.Errorf("%w: ws unit list prints the units", err)}
	}
	return units, err
}

// refuse says why a command that changes or reads root's paths does not run here, or nil where it
// does.
func (s UnitSide) refuse(p Env) error {
	if s.Root && p.Euid != 0 {
		return errors.New("run it as root")
	}
	if s.Platform != nil {
		return s.Platform(p.Machine)
	}
	return nil
}

// SideRoles returns the specs' root with the roles that lack file left out of its listing, so that
// kind.Load finds the specs of a side's roles alone. Empty file leaves none out.
func SideRoles(specs fs.FS, file string) fs.FS {
	if file == "" {
		return specs
	}
	return sideRoles{specs, file}
}

// sideRoles is a root of specs that lists the roles holding file alone.
type sideRoles struct {
	fs.FS
	file string
}

// ReadDir lists the root without the roles that lack the side's file.
func (r sideRoles) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(r.FS, name)
	if err != nil || name != "." {
		return entries, err
	}
	return slices.DeleteFunc(entries, func(e fs.DirEntry) bool {
		_, err := fs.Stat(r.FS, path.Join(e.Name(), r.file))
		return err != nil
	}), nil
}
