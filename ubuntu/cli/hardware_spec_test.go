package cli

import (
	"os"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/ubuntu/values"
)

// The roles that need hardware, a GPU or a board, are tried on the fake machine alone. No machine
// of the tests has that hardware to spare.

// pciDevices holds a directory per PCI device, as sysfs lists them.
const pciDevices = "/sys/bus/pci/devices"

// specUnits loads the roles' specs with the defaults' values on m, and returns each unit's step by
// its name.
func specUnits(t *testing.T, m *effecttest.Machine, out *effecttest.Out) map[string]step.Step {
	v, err := values.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	steps := map[string]step.Step{}
	for _, u := range loadSpecs(t, kind.Env{M: m, U: &effecttest.Units{}, Out: out, Data: v, Self: selfPath}) {
		steps[u.Name] = u.Step
	}
	return steps
}

// loadSpecs loads the roles' specs on e, and returns their units in the order they run.
func loadSpecs(t *testing.T, e kind.Env) []kind.Unit {
	units, err := kind.Load(e, os.DirFS(rolesDir))
	if err != nil {
		t.Fatalf("the specs: %v", err)
	}
	return units
}

// planned plans the unit name of steps, and returns its changes' headlines.
func planned(t *testing.T, steps map[string]step.Step, name string) ([]string, error) {
	s, ok := steps[name]
	if !ok {
		t.Fatalf("no unit %s", name)
	}
	p, err := s.Plan(t.Context())
	var got []string
	for _, c := range p.Changes {
		got = append(got, step.Headline(c))
	}
	return got, err
}

// withPCI gives the machine one more PCI device of the class and the vendor.
func withPCI(m *effecttest.Machine, slot, class, vendor string) {
	dev := pciDevices + "/" + slot
	m.Dirs[pciDevices], m.Dirs[dev] = 0o755, 0o755
	m.Files[dev+"/class"], m.Files[dev+"/vendor"] = []byte(class+"\n"), []byte(vendor+"\n")
}
