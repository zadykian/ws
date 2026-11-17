package cli

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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
	// Dir is the root of the side's specs, DIR/ROLE/units.toml, where WS_UNITS names none.
	Dir string
	// Values loads the side's values. It returns them with the machine that the units run on,
	// which the values may confine.
	Values func(m effect.Machine) (data any, confined effect.Machine, err error)
	// Units is the service manager's units.
	Units effect.Units
	// Root refuses read and write to any user but root, as the side's units write root's paths.
	Root bool
	// System is the side's operating system, which decides how the kinds find a home and run a
	// command as a user.
	System kind.System
}

// unitRun is one call of ws unit: the units of the specs, in the order they run, and ws's lines,
// which go to stderr.
type unitRun struct {
	all  []kind.Unit
	term *effect.Terminal
	out  engine.Lines
}

// Unit returns ws unit: the units of ws's kinds, KIND/NAME, which a tool such as mise plans and
// applies through it.
func Unit(p Env, s UnitSide) *cobra.Command {
	return Group("unit", "plan and apply the units of ws's kinds, KIND/NAME, for mise",
		unitList(p, s), unitRead(p, s), unitWrite(p, s))
}

// unitList returns ws unit list.
func unitList(p Env, s UnitSide) *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "print each unit, KIND/NAME, in the order the units run",
		Long: "The specs are each units.toml one level below WS_UNITS, or below " + s.Dir + " where it is " +
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

// loadUnits loads the side's values and every unit of its specs. A spec that fails to load fails
// the command with status 1, as the specs and the values are the machine's, not the command line's.
func loadUnits(c *cobra.Command, p Env, s UnitSide) (unitRun, error) {
	term := effect.NewTerminal(strings.NewReader(""), c.ErrOrStderr())
	r := unitRun{term: term, out: engine.Lines{Out: term, Prog: c.CommandPath(), Seen: map[string]bool{}}}
	data, m, err := s.Values(p.Machine)
	if err != nil {
		return r, err
	}
	dir := cmp.Or(p.Getenv("WS_UNITS"), s.Dir)
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(p.Pwd, dir)
	}
	specs := os.DirFS(dir)
	// fs.Glob passes over a directory it cannot read, which would give no unit and no error.
	if _, err := fs.Stat(specs, "."); err != nil {
		return r, fmt.Errorf("the specs: %w", err)
	}
	r.all, err = kind.Load(kind.Env{M: m, U: s.Units, Out: r.out, Data: data, System: s.System}, specs)
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
	return nil
}
