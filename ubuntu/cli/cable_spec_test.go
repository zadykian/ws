package cli

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/ubuntu/values"
)

const (
	cableFile = "/etc/netplan/60-cable.yaml"
	// cableCheck is the file unit's check of the new file.
	cableCheck = selfPath + " ubuntu host cable check " + cableFile + ".ws-check"
	// cableBytes is the file that the step rendered for enp5s0 at 10.77.0.1/30.
	cableBytes = `# Written by github.com/zadykian/ws, whose next run replaces any change made here.
#
# The cable to the Mac: an address, and no DHCP, gateway or DNS. netplan merges this entry
# with other files' entries of the same name, as the installer's. Its settings win, as this file's
# name sorts after theirs, and its address joins theirs. An entry of another name that fits the
# interface keeps a networkd file of its own, which networkd may take first, so ws refuses one.
# optional: boot doesn't wait for the link, which is up only while the Mac is plugged in.
network:
  version: 2
  ethernets:
    "enp5s0":
      dhcp4: false
      dhcp6: false
      addresses: ["10.77.0.1/30"]
      optional: true
`
)

// cableUnits loads the host's specs for the values in yml on m, and returns their units in the order they
// run, with what they say on out.
func cableUnits(t *testing.T, m *effecttest.Machine, yml string) ([]kind.Unit, *effecttest.Out) {
	v, err := values.ParseHost([]byte(yml))
	if err != nil {
		t.Fatal(err)
	}
	out := &effecttest.Out{}
	return loadSpecs(t, kind.Env{M: m, U: &effecttest.Units{}, Out: out, Data: v, Side: "host", Self: selfPath}), out
}

// cableMachine is a machine with the cable's port.
func cableMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Dirs["/sys/class/net/enp5s0"] = 0o755
	m.Results[cableCheck] = effecttest.Result{}
	return m
}

const cableValues = "---\nWS_CABLE_PORT: enp5s0\n"

// applyUnit plans the unit name of units and applies its changes.
func applyUnit(t *testing.T, units []kind.Unit, name string) []string {
	i := slices.IndexFunc(units, func(u kind.Unit) bool { return u.Name == name })
	if i < 0 {
		t.Fatalf("no unit %s", name)
	}
	p, err := units[i].Step.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range p.Changes {
		got = append(got, step.Headline(c))
		if err := c.(interface{ Apply(context.Context) error }).Apply(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	return got
}

func TestCableSpecWritesTodaysBytes(t *testing.T) {
	m := cableMachine()
	units, _ := cableUnits(t, m, cableValues)
	applyUnit(t, units, "file/cable")
	if got := string(m.Files[cableFile]); got != cableBytes {
		t.Errorf("the file:\n%s\nwant:\n%s", got, cableBytes)
	}
	if !slices.Contains(m.Calls, "root "+cableCheck) {
		t.Errorf("calls %q, want the new file checked by ws", m.Calls)
	}
}

func TestCableFileIsRootsAnd600(t *testing.T) {
	m := cableMachine()
	units, _ := cableUnits(t, m, cableValues)
	applyUnit(t, units, "file/cable")
	if mode, owner := m.Modes[cableFile], m.Owners[cableFile]; mode != 0o600 || owner != [2]int{} {
		t.Errorf("mode %o, owner %v; want root's at 600", mode, owner)
	}
}

func TestCableSpecIsAbsentWithoutAPort(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/etc/netplan"] = 0o755
	m.Files[cableFile] = []byte(cableBytes)
	units, _ := cableUnits(t, m, "---\n")
	if got := applyUnit(t, units, "file/cable"); len(got) != 1 {
		t.Errorf("plan %q, want the file's removal", got)
	}
	if _, left := m.Files[cableFile]; left {
		t.Error("the file stays")
	}
}

func TestCableSpecIsOffWithoutTheInterface(t *testing.T) {
	m := effecttest.NewMachine()
	units, out := cableUnits(t, m, cableValues)
	for _, name := range []string{"guard/cable", "file/cable"} {
		if got := applyUnit(t, units, name); got != nil {
			t.Errorf("%s: plan %q, want none", name, got)
		}
	}
	if len(out.Lines) != 2 || !strings.Contains(out.Lines[0], "enp5s0") {
		t.Errorf("said %q, want each unit off for enp5s0", out.Lines)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "root ") }) {
		t.Errorf("calls %q, want no command", m.Calls)
	}
}

func TestCableFileWaitsOnTheGuard(t *testing.T) {
	units, _ := cableUnits(t, cableMachine(), cableValues)
	var names []string
	for _, u := range units {
		names = append(names, u.Name)
	}
	guard, file, reload := slices.Index(names, "guard/cable"), slices.Index(names, "file/cable"), slices.Index(names, "state/cable-reload")
	if guard < 0 || file < guard || reload < file {
		t.Errorf("units %q, want the guard, the file, then the reload", names)
	}
}

func TestCableGuardRunsTheCheck(t *testing.T) {
	m := cableMachine()
	m.Results[selfPath+" ubuntu host cable check"] = effecttest.Result{}
	units, _ := cableUnits(t, m, cableValues)
	applyUnit(t, units, "guard/cable")
	if !slices.Contains(m.Calls, "root "+selfPath+" ubuntu host cable check") {
		t.Errorf("calls %q, want ws's check as root", m.Calls)
	}
}
