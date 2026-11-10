package nvidia

import (
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

// checkLine is the simulated install of every name the role pins.
var checkLine = "apt-get install -s " + strings.Join(allNames(), " ")

// withDevice returns a machine with one PCI device of the class and the vendor.
func withDevice(class, vendor string) *effecttest.Machine {
	m := effecttest.NewMachine()
	dev := pciDevices + "/0000:01:00.0"
	m.Dirs[pciDevices], m.Dirs[dev] = 0o755, 0o755
	m.Files[dev+"/class"], m.Files[dev+"/vendor"] = []byte(class+"\n"), []byte(vendor+"\n")
	return m
}

func TestHasGPU(t *testing.T) {
	for _, tc := range []struct {
		name string
		m    *effecttest.Machine
		want bool
	}{
		{"NVIDIA's display device", withDevice("0x030000", "0x10de"), true},
		{"NVIDIA's 3D controller", withDevice("0x030200", "0x10de"), true},
		{"NVIDIA's audio device", withDevice("0x040300", "0x10de"), false},
		{"virtio's display device", withDevice("0x030000", "0x1af4"), false},
		{"no PCI devices", effecttest.NewMachine(), false},
	} {
		if got, err := hasGPU(tc.m); err != nil || got != tc.want {
			t.Errorf("%s: hasGPU = %v, %v; want %v", tc.name, got, err, tc.want)
		}
	}
}

func TestGPUChecksTheNamesWithoutOne(t *testing.T) {
	m, out := withDevice("0x030000", "0x1af4"), &effecttest.Out{}
	m.Results[checkLine] = effecttest.Result{}
	if changes, err := (gpu{m, out}).Plan(t.Context()); err != nil || len(changes) != 0 {
		t.Errorf("Plan = %v, %v; want none", changes, err)
	}
	if len(out.Lines) != 1 || !strings.Contains(out.Lines[0], "no NVIDIA GPU") {
		t.Errorf("says %q", out.Lines)
	}
	m.Results[checkLine] = effecttest.Result{Err: errors.New("E: Unable to locate package")}
	if _, err := (gpu{m, out}).Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "Unable to locate") {
		t.Errorf("Plan with a name gone = %v; want apt's error", err)
	}
}

func TestGPUChecksNothingWithOne(t *testing.T) {
	m := withDevice("0x030000", "0x10de")
	if changes, err := (gpu{m, &effecttest.Out{}}).Plan(t.Context()); err != nil || len(changes) != 0 {
		t.Errorf("Plan = %v, %v; want none and no apt-get", changes, err)
	}
}
