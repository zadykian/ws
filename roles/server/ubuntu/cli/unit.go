package cli

import (
	"context"
	"errors"
	"io/fs"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/local"
	"github.com/zadykian/ws/engine/core/effect/units"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// unitsDir is the root of the server's specs, a units.toml in each role's directory, in the
// checkout that mise bootstrap --from-dir /opt/ws makes.
const unitsDir = "/opt/ws/roles/server/ubuntu"

// UnitCmd returns ws unit with the server's side: its specs, its values and systemd's units. Its
// units write root's paths, so read and write run as root.
func UnitCmd(p enginecli.Env) *cobra.Command { return enginecli.Unit(p, unitSide(serverVars)) }

// unitSide returns the server's side of ws unit, with its values read from vars.
func unitSide(vars string) enginecli.UnitSide {
	return enginecli.UnitSide{Dir: unitsDir, Units: units.OSUnits{}, Root: true,
		Values: func(_ context.Context, m effect.Machine) (any, effect.Machine, error) {
			v, err := values.Load(m, vars)
			if err != nil {
				return nil, nil, err
			}
			// WS_HOME is known now: the machine's writes under it stay inside it, as apply's do.
			if om, ok := m.(local.OSMachine); ok {
				om.Confine = v.Home
				m = om
			}
			return v, m, nil
		},
		Secrets: loadSecretsFile}
}

// loadSecretsFile loads ws's secrets file for /etc/ws/env. A machine without one, as in CI, has no
// secrets, and its secret files stay unchanged.
func loadSecretsFile(ctx context.Context, m effect.Machine) (any, error) {
	s, err := secrets.Store{M: m}.Load(ctx)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, kind.ErrNoSecrets
	}
	return s, err
}
