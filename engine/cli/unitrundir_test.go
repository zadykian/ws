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

// runUnitAt runs ws unit with args on a side of fixed values, whose run directory is runDir, or
// the specs' parent where that is empty.
func runUnitAt(e cli.Env, runDir string, args ...string) (status int, stdout, stderr string) {
	side := cli.UnitSide{
		Dir: "/nowhere", RunDir: runDir,
		Values: func(context.Context, effect.Machine) (any, error) {
			return unitValues{User: "u"}, nil
		},
		Units: &effecttest.Units{},
		Root:  true,
	}
	return clitest.Run(e, []*cobra.Command{cli.Unit(e, side)}, "", append([]string{"unit"}, args...)...)
}

func TestUnitReadWholeRunsInTheDirectoryTheSideNames(t *testing.T) {
	e, _ := unitEnv(t, map[string]string{
		"r/units.toml.tmpl": fileUnit("a", "/etc/a", ""), "r/a": "1\n",
	})
	// The specs' parent is no run directory here: the side's is a sibling that holds the link.
	runDir := filepath.Join(filepath.Dir(filepath.Dir(e.Getenv("WS_UNITS"))), "guest")
	e.Pwd = filepath.Dir(e.Getenv("WS_UNITS"))
	want := "file/a\tfails: mise runs in " + e.Pwd + ", which loads none of the specs' lines: run mise in " + runDir + "\n"
	if status, stdout, _ := runUnitAt(e, runDir, "read", "--whole", "file/a"); status != 0 || stdout != want {
		t.Errorf("in the specs' parent: status = %d, stdout = %q; want %q", status, stdout, want)
	}
	e.Pwd = runDir
	if status, stdout, _ := runUnitAt(e, runDir, "read", "--whole", "file/a"); status != 0 || strings.Contains(stdout, "fails") {
		t.Errorf("in the run directory: status = %d, stdout = %q; want the unit read", status, stdout)
	}
}

func TestUnitVerifyRunsInTheDirectoryTheSideNames(t *testing.T) {
	e, _ := unitEnv(t, map[string]string{
		"r/units.toml.tmpl": fileUnit("a", "/etc/a", ""), "r/a": "1\n",
	})
	runDir := filepath.Join(filepath.Dir(filepath.Dir(e.Getenv("WS_UNITS"))), "guest")
	e.Pwd = filepath.Dir(e.Getenv("WS_UNITS"))
	want := "mise runs in " + e.Pwd + ", which loads none of the specs' lines: run mise in " + runDir
	status, stdout, stderr := runUnitAt(e, runDir, "verify")
	if status != 1 || stdout != "" || !strings.Contains(stderr, want) || strings.Contains(stderr, "file/a") {
		t.Errorf("in the specs' parent: status = %d, stdout = %q, stderr = %q; want %q alone", status, stdout, stderr, want)
	}
}
