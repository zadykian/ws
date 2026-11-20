package cli_test

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// A plan, which writes no input's file, reads the input from the run's env, as the run writes it.
func TestUnitReadTakesAnInputOfTheRun(t *testing.T) {
	specs := map[string]string{
		"r/units.toml.tmpl": "[file.a]\npath = \"/etc/a\"\ntemplate = \"a.tmpl\"\n",
		"r/a.tmpl":          `{{ envvar "/etc/env" "IN" }}` + "\n",
	}
	for _, c := range []struct {
		env  map[string]string
		want string
	}{
		{map[string]string{}, "file/a\tin-sync\n"},
		{map[string]string{"IN": "old"}, "file/a\tin-sync\n"},
		{map[string]string{"IN": "new"}, "file/a\twrite /etc/a"},
	} {
		e, m := unitEnv(t, specs)
		c.env["WS_UNITS"] = e.Getenv("WS_UNITS")
		e = clitest.WithEnv(e, c.env)
		m.Files["/etc/env"], m.Files["/etc/a"] = []byte("IN=old\n"), []byte("old\n")
		side := cli.UnitSide{
			Dir: "/nowhere",
			Values: func(_ context.Context, m effect.Machine) (any, effect.Machine, error) {
				return unitValues{User: "u"}, m, nil
			},
			Units: &effecttest.Units{}, Root: true, Hidden: []string{"IN"},
		}
		status, stdout, stderr := clitest.Run(e, []*cobra.Command{cli.Unit(e, side)}, "", "unit", "read", "file/a")
		if status != 0 || !strings.HasPrefix(stdout, c.want) {
			t.Errorf("IN %q: status = %d, stdout = %q, stderr = %q; want %q", c.env["IN"], status, stdout, stderr, c.want)
		}
		if strings.Contains(stdout+stderr, "new") {
			t.Errorf("IN %q: the output holds the value: %q, %q", c.env["IN"], stdout, stderr)
		}
	}
}
