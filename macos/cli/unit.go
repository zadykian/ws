package cli

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/macos/facts"
)

// unitsDir is the root of the Mac's specs, a units.toml.tmpl in each role's directory. It sits in the
// checkout at /opt/ws, where mise -C /opt/ws/macos bootstrap runs.
const unitsDir = "/opt/ws/macos/roles"

// UnitCmd returns ws unit with the Mac's side: its specs and its facts. It runs as the user, as
// mise does on the Mac, and a change to root's paths goes through sudo. p's machine reads root's
// paths as the user, so that a read asks no password.
func UnitCmd(p enginecli.Env) *cobra.Command { return enginecli.Unit(p, unitSide(p)) }

// unitSide returns the Mac's side of ws unit, with its facts read from p's environment.
func unitSide(p enginecli.Env) enginecli.UnitSide {
	return enginecli.UnitSide{Dir: unitsDir, System: kind.MacOS,
		Values: func(ctx context.Context, m effect.Machine) (any, error) {
			if p.Euid == 0 {
				// As root, a unit's file of the home would be root's, and its read would ask nothing.
				return nil, errors.New("run it as yourself, not root: it asks sudo where it needs root")
			}
			f, err := facts.Read(ctx, m, p.Getenv, p.Pwd)
			if err != nil {
				return nil, err
			}
			return f, nil
		},
		User: func(data any) string { return data.(facts.Mac).User }}
}
