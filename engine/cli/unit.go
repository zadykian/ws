package cli

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

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
	// Dir is the root of the side's specs, DIR/ROLE/units.toml.tmpl, where WS_UNITS names none.
	Dir string
	// Values loads the side's values. It returns them with the machine that the units run on,
	// which the values may confine.
	Values func(ctx context.Context, m effect.Machine) (data any, confined effect.Machine, err error)
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
	// Secrets loads the side's secrets for a secret file's template, or returns kind.ErrNoSecrets
	// where the machine has none. Nil is a side with no secrets.
	Secrets func(ctx context.Context, m effect.Machine) (any, error)
	// User returns the account that ws runs as, from the values, where it runs as a user: the
	// Mac's. A run state of that user's needs no sudo there. Nil is a side that runs as root.
	User func(data any) string
	// Hidden names the variables of the run's env that hold the side's secrets, which a run state
	// drops from each command but ws's (kind.Env.Hidden).
	Hidden []string
}

// unitRun is one call of ws unit: the units of the specs, in the order they run, ws's lines,
// which go to stderr, and the specs' directory.
type unitRun struct {
	all  []kind.Unit
	term *effect.Terminal
	out  engine.Lines
	dir  string
}

// runDir is the directory mise runs in for the side: the parent of the specs, whose mise/conf.d
// links them. A run elsewhere loads none of their lines, so a failure for a line names it.
func (r unitRun) runDir() string {
	return filepath.Dir(filepath.Clean(r.dir))
}

// Unit returns ws unit: the units of ws's kinds, KIND/NAME, which a tool such as mise plans and
// applies through it.
func Unit(p Env, s UnitSide) *cobra.Command {
	return Group("unit", "plan and apply the units of ws's kinds, KIND/NAME, for mise",
		unitList(p, s), unitRead(p, s), unitWrite(p, s), unitVerify(p, s))
}

// unitList returns ws unit list.
func unitList(p Env, s UnitSide) *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "print each unit, KIND/NAME, in the order the units run",
		Long: "The specs are each units.toml.tmpl one level below WS_UNITS, or below " + s.Dir + " where it is " +
			"unset. --toml prints each unit as a line of mise's [bootstrap.packages].",
		Args: cobra.NoArgs,
	}
	asTOML := c.Flags().Bool("toml", false, "print the units as lines of mise's [bootstrap.packages]")
	c.RunE = func(c *cobra.Command, _ []string) error {
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
	term := effect.NewTerminal(strings.NewReader(""), c.ErrOrStderr())
	r := unitRun{term: term, out: engine.Lines{Out: term, Prog: c.CommandPath(), Seen: map[string]bool{}}}
	data, m, err := s.Values(c.Context(), p.Machine)
	if err != nil {
		return r, err
	}
	dir := cmp.Or(p.Getenv("WS_UNITS"), s.Dir)
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(p.Pwd, dir)
	}
	r.dir = dir
	specs := os.DirFS(dir)
	// fs.Glob passes over a directory it cannot read, which would give no unit and no error.
	if _, err := fs.Stat(specs, "."); err != nil {
		return r, fmt.Errorf("the specs: %w", err)
	}
	self, err := p.Self()
	if err != nil {
		return r, fmt.Errorf("finding the file ws runs from: %w", err)
	}
	env := kind.Env{M: m, U: s.Units, Out: r.out, Data: data, System: s.System, Self: self, Hidden: s.Hidden}
	if s.Secrets != nil {
		env.Secrets = loadOnce(s.Secrets, m)
	}
	if s.User != nil {
		env.User = s.User(data)
	}
	r.all, err = kind.Load(env, specs)
	if err != nil {
		return r, fmt.Errorf("%s: %w", dir, err)
	}
	return r, nil
}

// loadOnce returns load as kind.Env takes it, run once a command at most, and only where a unit
// calls it: a load decrypts the identity through systemd-creds.
func loadOnce(load func(context.Context, effect.Machine) (any, error), m effect.Machine) func(context.Context) (any, error) {
	var once sync.Once
	var secrets any
	var err error
	return func(ctx context.Context) (any, error) {
		once.Do(func() { secrets, err = load(ctx, m) })
		return secrets, err
	}
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
