package cli

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/units"
	"github.com/zadykian/ws/ubuntu/values"
)

// osRelease names the machine's system, which the server's units are written for.
const osRelease = "/etc/os-release"

// unitsDir is the root of the specs, a units.toml.tmpl in each role's directory, in the checkout
// that ws ubuntu guest setup makes at /opt/ws. runDir is where mise runs for the guest, which holds
// the link to the roles, its lock and its tasks.
const (
	unitsDir = "/opt/ws/ubuntu/roles"
	runDir   = "/opt/ws/ubuntu/guest"
)

// UnitCmd returns ws unit with the server's side: its specs, its values and systemd's units. Its
// units write root's paths for Ubuntu 26.04's packages and services, so read and write run as root,
// and on Ubuntu 26.04 alone.
func UnitCmd(p enginecli.Env) *cobra.Command { return enginecli.Unit(p, unitSide(guestVars)) }

// unitSide returns the server's side of ws unit, with its values read from vars.
func unitSide(vars string) enginecli.UnitSide {
	return enginecli.UnitSide{Dir: unitsDir, RunDir: runDir, Units: units.OSUnits{}, Root: true, Platform: ubuntu,
		Values: func(_ context.Context, m effect.Machine) (any, error) {
			return values.Load(m, vars)
		},
		Hidden: secretInputs, Task: secretsTask}
}

// ubuntu refuses a machine whose /etc/os-release names a system other than Ubuntu 26.04, the one
// that the roles' packages, paths and services are written for. A machine with no such file is
// refused too.
func ubuntu(m effect.Machine) error {
	text, err := m.ReadFile(osRelease)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	var id, version string
	sc := bufio.NewScanner(bytes.NewReader(text))
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), "=")
		if !ok {
			continue
		}
		v = strings.Trim(v, `"'`)
		switch k {
		case "ID":
			id = v
		case "VERSION_ID":
			version = v
		}
	}
	if id != "ubuntu" || version != "26.04" {
		return fmt.Errorf("the server's units are written for Ubuntu 26.04 alone, not %q %q", id, version)
	}
	return nil
}
