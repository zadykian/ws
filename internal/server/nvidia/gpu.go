package nvidia

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
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

// gpu checks, on a machine without NVIDIA's GPU, that apt finds every package the role pins, so
// that CI's run fails where the archive drops one. It installs nothing.
type gpu struct {
	m   effect.Machine
	out effect.Out
}

func (g gpu) Plan(ctx context.Context) ([]host.Change, error) {
	ok, err := hasGPU(g.m)
	if err != nil || ok {
		return nil, err
	}
	if _, err := g.m.Run(ctx, "apt-get", append([]string{"install", "-s"}, allNames()...)...); err != nil {
		return nil, fmt.Errorf("nvidia: apt finds not every package the role pins: %w", err)
	}
	g.out.Say("nvidia: this machine has no NVIDIA GPU, so the role installs no driver")
	return nil, nil
}
