package cli

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
)

const incusPoolCommand = asRoot + selfPath + " ubuntu host incus pool"

// incusUnits loads the host's units for the default values.
func incusUnits(t *testing.T, m *effecttest.Machine) []kind.Unit {
	units, _ := cableUnits(t, m, "---\n")
	return units
}

func TestIncusPoolRunsWhereItsCheckFails(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[incusPoolCommand+" --check"] = effecttest.Result{Err: errors.New("to do: create the storage pool ws")}
	m.Results[incusPoolCommand] = effecttest.Result{}
	got := applyUnit(t, incusUnits(t, m), "state/incus-pool")
	if len(got) != 1 || !strings.Contains(got[0], selfPath+" ubuntu host incus pool") {
		t.Errorf("plan %q, want the pool's command", got)
	}
	if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasSuffix(c, incusPoolCommand) && !strings.HasSuffix(c, "--check") }) {
		t.Errorf("calls %q, want the command run as root", m.Calls)
	}
}

func TestIncusPoolIsInSyncWhereItsCheckPasses(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[incusPoolCommand+" --check"] = effecttest.Result{}
	if got := applyUnit(t, incusUnits(t, m), "state/incus-pool"); got != nil {
		t.Errorf("plan %q, want none", got)
	}
}
