package cli

import (
	"os"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
)

// before fails the test where a unit of first does not run ahead of every unit of then, in the
// units' order.
func before(t *testing.T, first string, then ...string) {
	names := specLines(t, defaults(t))
	i := slices.Index(names, first)
	for _, n := range then {
		if j := slices.Index(names, n); i < 0 || j < 0 || i > j {
			t.Errorf("%s at %d, %s at %d; want %s first", first, i, n, j, first)
		}
	}
}

// The collector's units run after signoz's units, so that SigNoz's retention is set before the
// collector sends. The endpoint they read is in /etc/ws/env, which mise writes before any unit.
func TestOtelcolRunsAfterSignoz(t *testing.T) {
	before(t, "state/signoz-password", "deb/otelcol")
	before(t, "state/signoz-sync", "deb/otelcol")
	before(t, "deb/otelcol", "service/otelcol")
}

// A read takes a unit as pending where a unit it waits on, by its chain of after, has changes. The
// collector's service starts after SigNoz's sync, so its chain holds it.
func TestOtelcolWaitsOnSignoz(t *testing.T) {
	e := kind.Env{M: effecttest.NewMachine(), U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: defaults(t), Self: selfPath}
	units, err := kind.Load(e, os.DirFS(rolesDir))
	if err != nil {
		t.Fatal(err)
	}
	chain, err := kind.Closure(units, []string{"service/otelcol"})
	if err != nil {
		t.Fatal(err)
	}
	for _, w := range []string{"deb/otelcol", "state/signoz-sync"} {
		if !slices.Contains(kind.Names(chain), w) {
			t.Errorf("service/otelcol: its chain %q lacks %s", kind.Names(chain), w)
		}
	}
}

// SigNoz's project runs on the system's daemon, which docker's units set up.
func TestSignozRunsAfterDocker(t *testing.T) {
	before(t, "service/docker", "compose/signoz")
}

// signoz's units run in order: the password, the project, then the sync.
func TestSignozUnitsRunInOrder(t *testing.T) {
	before(t, "state/signoz-password", "compose/signoz")
	before(t, "compose/signoz", "state/signoz-sync")
}
