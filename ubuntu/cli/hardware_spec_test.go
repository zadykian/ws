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

// The roles that need hardware, a GPU or a board, are tried on the fake machine alone. No machine
// of the tests has that hardware to spare.

// pciDevices holds a directory per PCI device, as sysfs lists them.
const pciDevices = "/sys/bus/pci/devices"

// specUnits loads the guest's specs with the defaults' values on m, and returns each unit's step by
// its name.
func specUnits(t *testing.T, m *effecttest.Machine, out *effecttest.Out) map[string]step.Step {
	v, err := values.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	steps := map[string]step.Step{}
	for _, u := range loadSpecs(t, kind.Env{M: m, U: &effecttest.Units{}, Out: out, Data: v, Side: "guest", Self: selfPath}) {
		steps[u.Name] = u.Step
	}
	return steps
}

// hostSteps is specUnits for the host: its specs over the host's default values.
func hostSteps(t *testing.T, m *effecttest.Machine, out *effecttest.Out) map[string]step.Step {
	v, err := values.ParseHost(nil)
	if err != nil {
		t.Fatal(err)
	}
	steps := map[string]step.Step{}
	for _, u := range loadSpecs(t, kind.Env{M: m, U: &effecttest.Units{}, Out: out, Data: v, Side: "host", Self: selfPath}) {
		steps[u.Name] = u.Step
	}
	return steps
}

// guestUnits loads the guest's specs for the values in yml on m, and returns their units in the
// order they run, with what they say on out.
func guestUnits(t *testing.T, m *effecttest.Machine, yml string) ([]kind.Unit, *effecttest.Out) {
	v, err := values.Parse([]byte(yml))
	if err != nil {
		t.Fatal(err)
	}
	out := &effecttest.Out{}
	return loadSpecs(t, kind.Env{M: m, U: &effecttest.Units{}, Out: out, Data: v, Side: "guest", Self: selfPath}), out
}

// loadSpecs loads the specs of the roles that hold the mise file of e's side, on e, and returns
// their units in the order they run.
func loadSpecs(t *testing.T, e kind.Env) []kind.Unit {
	units, err := kind.Load(e, enginecli.SideRoles(os.DirFS(rolesDir), "mise."+e.Side+".toml"))
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
