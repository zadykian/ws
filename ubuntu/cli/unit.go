package cli

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/units"
	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/ubuntu/values"
)

// osRelease names the machine's system, which the server's units are written for.
const osRelease = "/etc/os-release"

// unitsDir is the root of the specs, a units.toml.tmpl in each role's directory, in the checkout
// that ws ubuntu guest setup makes at /opt/ws. runDir is where mise runs for the guest, which holds
// the link to the roles, its lock and its tasks. hostRunDir is the host's, which holds the link
// alone: the host pins no tool and has no secret.
const (
	unitsDir   = "/opt/ws/ubuntu/roles"
	runDir     = "/opt/ws/ubuntu/guest"
	hostRunDir = "/opt/ws/ubuntu/host"
)

// UnitCmd returns ws unit with the server's two sides, the guest's and the host's: their specs,
// their values and systemd's units. Each runs in its run directory, and serves the roles that hold
// its mise file. The units write root's paths for Ubuntu 26.04's packages and services, so read and
// write run as root, and on Ubuntu 26.04 alone.
func UnitCmd(p enginecli.Env) *cobra.Command {
	return enginecli.Unit(p, unitSide(guestVars), hostSide(hostVars))
}

// hostSide returns the host's side of ws unit, with its values read from vars.
func hostSide(vars string) enginecli.UnitSide {
	return enginecli.UnitSide{Name: "host", RoleFile: "mise.host.toml", Dir: unitsDir, RunDir: hostRunDir,
		Units: units.OSUnits{}, Root: true, Platform: ubuntu,
		Values: func(_ context.Context, m effect.Machine) (any, error) {
			return values.LoadHost(m, vars)
		}}
}

// unitSide returns the server's side of ws unit, with its values read from vars.
func unitSide(vars string) enginecli.UnitSide {
	return enginecli.UnitSide{Name: "guest", RoleFile: "mise.guest.toml", Dir: unitsDir, RunDir: runDir,
		Units: units.OSUnits{}, Root: true, Platform: ubuntu,
		Values: func(_ context.Context, m effect.Machine) (any, error) {
			return unitValues(m, vars)
		},
		Hidden: secretInputs, Task: secretsTask}
}

// unitValues reads the guest's values, with the mode that its miserc names. Mise's files follow the
// miserc, which setup writes from the values, so the units follow it too. A values edit takes
// effect at the next setup, and a bootstrap between runs the files and the units of one mode.
func unitValues(m effect.Machine, vars string) (values.Values, error) {
	v, err := values.Load(m, vars)
	if err != nil {
		return v, err
	}
	envs, err := setup.SideEnvs(m)
	v.SelfContained = slices.Contains(envs, setup.SelfContained)
	return v, err
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
