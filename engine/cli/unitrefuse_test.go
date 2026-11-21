package cli_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestUnitReadNeedsRoot(t *testing.T) {
	e, _ := unitEnv(t, map[string]string{"r/units.toml.tmpl": fileUnit("a", "/etc/a", "")})
	e.Euid = 1000
	for _, verb := range []string{"read", "write"} {
		if status, _, stderr := runUnit(e, verb, "file/a"); status != 1 || !strings.Contains(stderr, "run it as root") {
			t.Errorf("%s: status = %d, stderr = %q", verb, status, stderr)
		}
	}
}

func TestUnitPlatformRefusesReadAndWrite(t *testing.T) {
	e, m := unitEnv(t, map[string]string{"r/units.toml.tmpl": fileUnit("a", "/etc/a", ""), "r/a": "1\n"})
	side := cli.UnitSide{
		Dir: "/nowhere",
		Values: func(context.Context, effect.Machine) (any, error) {
			return unitValues{User: "u"}, nil
		},
		Units:    &effecttest.Units{},
		Root:     true,
		Platform: func(effect.Machine) error { return errors.New("not the system") },
	}
	for _, verb := range []string{"read", "write"} {
		status, _, stderr := clitest.Run(e, []*cobra.Command{cli.Unit(e, side)}, "", "unit", verb, "file/a")
		if status != 1 || !strings.Contains(stderr, "not the system") {
			t.Errorf("%s: status = %d, stderr = %q", verb, status, stderr)
		}
	}
	if _, wrote := m.Files["/etc/a"]; wrote {
		t.Error("a refused write wrote file/a")
	}
}
