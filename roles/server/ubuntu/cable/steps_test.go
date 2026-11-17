package cable

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/cable/internal/netplan"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// rig is a booted machine with the cable's port, and the step on it.
type rig struct {
	m     *effecttest.Machine
	u     *effecttest.Units
	out   *effecttest.Out
	steps []step.Step
}

func newRig(iface, address string) rig {
	m, u := portMachine()
	m.Results["netplan generate"] = effecttest.Result{}
	m.Results["networkctl reload"] = effecttest.Result{}
	v := values.Default()
	v.Cable.Port, v.Cable.Address = iface, address
	out := &effecttest.Out{}
	return rig{m, u, out, Steps(v, secrets.Secrets{}, m, u, out)}
}

func (r rig) run(t *testing.T, check bool) (bool, error) {
	return step.Run(t.Context(), r.steps, r.out, check)
}

func TestTheFileLandsAndNetworkdReloads(t *testing.T) {
	r := newRig("enp5s0", "10.77.0.1/30")
	changed, err := r.run(t, false)
	if err != nil || !changed {
		t.Fatalf("changed = %v, err = %v", changed, err)
	}
	if got := string(r.m.Files[netplan.Path]); !strings.Contains(got, `addresses: ["10.77.0.1/30"]`) {
		t.Errorf("the file:\n%s", got)
	}
	if mode := r.m.Modes[netplan.Path]; mode != 0o600 {
		t.Errorf("mode = %o, want 600", mode)
	}
	gen, rel := slices.Index(r.m.Calls, "root netplan generate"), slices.Index(r.m.Calls, "root networkctl reload")
	if gen < 0 || rel < gen {
		t.Errorf("want generate, then reload, in %q", r.m.Calls)
	}
	// step.Run planned again after the apply and found nothing; a third plan agrees.
	if changed, err := r.run(t, true); err != nil || changed {
		t.Errorf("a second run: changed = %v, err = %v", changed, err)
	}
}

func TestACleanMachineChangesNothing(t *testing.T) {
	r := newRig("enp5s0", "10.77.0.1/30")
	file, err := netplan.Render("enp5s0", "10.77.0.1/30")
	if err != nil {
		t.Fatal(err)
	}
	r.m.Files[netplan.Path] = file
	r.m.Modes[netplan.Path] = 0o600
	r.m.Root[netplan.Path] = effect.Attrs{Mode: 0o600}
	if changed, err := r.run(t, true); err != nil || changed {
		t.Errorf("changed = %v, err = %v", changed, err)
	}
}
