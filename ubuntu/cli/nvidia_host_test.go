package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestNvidiaHostWaitsForTheDeviceNodesBeforeIncus(t *testing.T) {
	m := withNvidia()
	steps := hostSteps(t, m, &effecttest.Out{})
	got, err := planned(t, steps, "service/nvidia-devices")
	if err != nil {
		t.Fatal(err)
	}
	all := strings.Join(got, "\n")
	for _, want := range []string{"/etc/systemd/system/ws-nvidia-devices.service", "/etc/systemd/system/incus.service.d/ws-nvidia.conf"} {
		if !strings.Contains(all, want) {
			t.Errorf("the plan lacks %s:\n%s", want, all)
		}
	}
	for _, name := range []string{"packages/nvidia-userspace", "file/nvidia-userspace-pin"} {
		if _, ok := steps[name]; ok {
			t.Errorf("the host holds the guest's unit %s", name)
		}
	}
	bare := hostSteps(t, effecttest.NewMachine(), &effecttest.Out{})
	if got, err := planned(t, bare, "service/nvidia-devices"); err != nil || got != nil {
		t.Errorf("without the card: plan = %q, %v; want nothing", got, err)
	}
}

func TestNvidiaDeviceNodesFilesOrderIncusAfterTheWait(t *testing.T) {
	unit, err := os.ReadFile(rolesDir + "/nvidia/files/ws-nvidia-devices.service")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/dev/nvidia0", "/dev/nvidia-uvm", "Type=oneshot", "RemainAfterExit=yes", "timeout 60"} {
		if !strings.Contains(string(unit), want) {
			t.Errorf("the unit lacks %q:\n%s", want, unit)
		}
	}
	if strings.ContainsAny(strings.SplitN(string(unit), "ExecStart=", 2)[1], "$%") {
		t.Errorf("the unit's command holds $ or %%, which systemd expands:\n%s", unit)
	}
	// At boot, udev loads the module as it handles the card's event, after the unit may start. A
	// check before udev's queue drains would skip the wait on a host whose module is about to load.
	command := strings.SplitN(string(unit), "ExecStart=", 2)[1]
	settle, gate := strings.Index(command, "udevadm settle --timeout=60;"), strings.Index(command, "test -d /sys/module/nvidia")
	if settle < 0 || gate < settle {
		t.Errorf("the unit reads /sys/module/nvidia before udevadm settle, bounded:\n%s", command)
	}
	drop, err := os.ReadFile(rolesDir + "/nvidia/files/incus-nvidia.conf")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"[Unit]", "Wants=ws-nvidia-devices.service", "After=ws-nvidia-devices.service"} {
		if !strings.Contains(string(drop), want) {
			t.Errorf("the drop-in lacks %q:\n%s", want, drop)
		}
	}
}
