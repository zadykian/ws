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

// tableSecrets stand in for a side's secrets.
type tableSecrets struct{ Env map[string]string }

// secretUnit is a secret file unit at /etc/NAME, whose template reads the secrets' table.
func secretUnit(name string) string {
	return "[file." + name + "]\npath = \"/etc/" + name + "\"\ntemplate = \"t\"\nsecret = true\n"
}

// runWithSecrets runs ws unit with args on a side whose secrets come from load.
func runWithSecrets(e cli.Env, load func(context.Context, effect.Machine) (any, error), args ...string) (int, string) {
	side := cli.UnitSide{
		Dir: "/nowhere",
		Values: func(_ context.Context, m effect.Machine) (any, effect.Machine, error) {
			return unitValues{User: "u"}, m, nil
		},
		Units:   &effecttest.Units{},
		Root:    true,
		Secrets: load,
	}
	status, _, stderr := clitest.Run(e, []*cobra.Command{cli.Unit(e, side)}, "", append([]string{"unit"}, args...)...)
	return status, stderr
}

func TestUnitSideSecretsReachTheSpec(t *testing.T) {
	e, m := unitEnv(t, map[string]string{"r/units.toml.tmpl": secretUnit("a"), "r/t": "A={{ index (secrets).Env \"A\" }}\n"})
	load := func(context.Context, effect.Machine) (any, error) {
		return tableSecrets{map[string]string{"A": "dummy"}}, nil
	}
	if status, stderr := runWithSecrets(e, load, "write", "file/a"); status != 0 || string(m.Files["/etc/a"]) != "A=dummy\n" {
		t.Errorf("status = %d, file = %q, stderr = %q", status, m.Files["/etc/a"], stderr)
	}
}

func TestUnitSideSecretsDecryptOnce(t *testing.T) {
	e, m := unitEnv(t, map[string]string{
		"r/units.toml.tmpl": secretUnit("a") + secretUnit("b") + fileUnit("c", "/etc/c", ""),
		"r/t":               "{{ (secrets).Env }}\n", "r/c": "3\n",
	})
	calls := 0
	load := func(_ context.Context, got effect.Machine) (any, error) {
		calls++
		if got != m {
			t.Error("the secrets were loaded on another machine")
		}
		return tableSecrets{}, nil
	}
	if status, stderr := runWithSecrets(e, load, "write", "file/a", "file/b"); status != 0 || calls != 1 {
		t.Errorf("status = %d, loads = %d, stderr = %q; want one load", status, calls, stderr)
	}
	calls = 0
	if status, stderr := runWithSecrets(e, load, "write", "file/c"); status != 0 || calls != 0 {
		t.Errorf("status = %d, loads = %d, stderr = %q; want none without a secret unit", status, calls, stderr)
	}
	if status, stderr := runWithSecrets(e, nil, "write", "--dry-run", "file/a"); status != 0 ||
		!strings.Contains(stderr, "/etc/a: left as it is: this machine holds no secrets file") {
		t.Errorf("a side with no secrets: status = %d, stderr = %q", status, stderr)
	}
}

func TestUnitReadSaysNoSecretsOnce(t *testing.T) {
	e, _ := unitEnv(t, map[string]string{"r/units.toml.tmpl": secretUnit("a"), "r/t": "{{ (secrets).Env }}\n"})
	if status, stderr := runWithSecrets(e, nil, "read", "file/a"); status != 0 ||
		strings.Count(stderr, "/etc/a: left as it is") != 1 {
		t.Errorf("read on a side with no secrets: status = %d, stderr = %q; want the line once", status, stderr)
	}
}
