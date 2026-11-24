package cli_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// sideValues stand in for a side's values: each side names itself, for a spec to read.
type sideValues struct{ Side string }

// twoSides returns the guest's and the host's sides over the specs, each with a run directory of
// its own. The Env's Pwd is the run directory of the side that in names, or the specs' parent.
func twoSides(t *testing.T, in string, specs map[string]string) (cli.Env, []cli.UnitSide) {
	e, _ := unitEnv(t, specs)
	var sides []cli.UnitSide
	for _, name := range []string{"guest", "host"} {
		dir := filepath.Join(filepath.Dir(e.Getenv("WS_UNITS")), name)
		sides = append(sides, cli.UnitSide{
			Dir: "/nowhere", RunDir: dir, Units: &effecttest.Units{}, Root: true,
			Values: func(context.Context, effect.Machine) (any, error) { return sideValues{Side: name}, nil },
		})
		if name == in {
			e.Pwd = dir
		}
	}
	return e, sides
}

func runSides(e cli.Env, sides []cli.UnitSide, args ...string) (status int, stdout, stderr string) {
	return clitest.Run(e, []*cobra.Command{cli.Unit(e, sides[0], sides[1:]...)}, "", append([]string{"unit"}, args...)...)
}

func TestUnitFillsASpecWithTheSidesValues(t *testing.T) {
	specs := map[string]string{"r/units.toml.tmpl": "[file.a]\npath = \"/etc/{{ .Side }}\"\nsource = \"a\"\n", "r/a": "1\n"}
	for _, side := range []string{"guest", "host"} {
		e, sides := twoSides(t, side, specs)
		if _, stdout, _ := runSides(e, sides, "read", "file/a"); stdout != "file/a\tcreate /etc/"+side+"\n" {
			t.Errorf("in the %s's run directory: read says %q", side, stdout)
		}
	}
}

func TestUnitInNoRunDirectoryNamesThem(t *testing.T) {
	e, sides := twoSides(t, "", map[string]string{"r/units.toml.tmpl": fileUnit("a", "/etc/a", ""), "r/a": "1\n"})
	dirs := sides[0].RunDir + " or " + sides[1].RunDir
	if status, _, stderr := runSides(e, sides, "list"); status != 1 || !strings.Contains(stderr, "no side's run directory: run mise in "+dirs) {
		t.Errorf("list: status = %d, stderr = %q; want the run directories named", status, stderr)
	}
}
