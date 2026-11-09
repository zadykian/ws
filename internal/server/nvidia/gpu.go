package nvidia

import (
	"errors"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/effect"
)

// pciDevices holds a directory per PCI device, with its class and vendor.
const pciDevices = "/sys/bus/pci/devices"

// hasGPU reports whether a PCI display device, of class 0x03, is NVIDIA's, of vendor 0x10de. A
// machine without sysfs's PCI devices has none.
func hasGPU(m effect.Machine) (bool, error) {
	names, err := m.ReadDir(pciDevices)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	for _, name := range names {
		class, err := m.ReadFile(pciDevices + "/" + name + "/class")
		if err != nil {
			return false, err
		}
		vendor, err := m.ReadFile(pciDevices + "/" + name + "/vendor")
		if err != nil {
			return false, err
		}
		if strings.HasPrefix(string(class), "0x03") && strings.TrimSpace(string(vendor)) == "0x10de" {
			return true, nil
		}
	}
	return false, nil
}
