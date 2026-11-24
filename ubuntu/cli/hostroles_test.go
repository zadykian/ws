package cli

import (
	"os"
	"testing"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/ubuntu/values"
)

// hostSpecs loads the roles' specs for the host, over the host's default values, and returns its
// units by name.
func hostSpecs(t *testing.T, m *effecttest.Machine) map[string]step.Step {
	v, err := values.ParseHost(nil)
	if err != nil {
		t.Fatal(err)
	}
	e := kind.Env{M: m, U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: v, Side: "host", Self: selfPath}
	units, err := kind.Load(e, enginecli.SideRoles(os.DirFS(rolesDir), "mise.host.toml"))
	if err != nil {
		t.Fatalf("the host's specs: %v", err)
	}
	steps := map[string]step.Step{}
	for _, u := range units {
		steps[u.Name] = u.Step
	}
	return steps
}
