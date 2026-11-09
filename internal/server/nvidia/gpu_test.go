package nvidia

import (
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

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
