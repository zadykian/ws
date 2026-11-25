package cli

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
)

const (
	moduleVersion = "/sys/module/nvidia/version"
	nvidiaPin     = "/etc/apt/preferences.d/ws-nvidia"
)

// userspaceNames are the userspace's packages. nvidia-smi's needs the other two at its own version,
// and apt picks their newest, so an install of an older one names each.
var userspaceNames = []string{nvidiaSMI, "libnvidia-compute-595-server", "nvidia-kernel-common-595-server"}

// userspaceAt is the install of the userspace at the module's upstream version v.
func userspaceAt(v string) string {
	pins := make([]string, len(userspaceNames))
	for i, n := range userspaceNames {
		pins[i] = n + "=" + v + "-*"
	}
	return "install " + strings.Join(pins, " ")
}

// userspaceInstalled is the userspace as apt installs it at the archive's revision of v.
func userspaceInstalled(m *effecttest.Machine, v string) {
	for _, n := range userspaceNames {
		m.Packages[n] = effecttest.Package{Version: v + "-0ubuntu0.26.04.2", Status: "ii"}
	}
}

// nvidiaUnits are the units of the role that a container's two modes turn on or off.
var nvidiaUnits = []string{"packages/nvidia", "state/nvidia-reboot", "packages/nvidia-userspace", "file/nvidia-userspace-pin"}

// containerWithCard is a container that shows its host's PCI devices, NVIDIA's card among them, and
// the module that the host loads.
func containerWithCard() *effecttest.Machine {
	m := withNvidia()
	m.Files["/run/systemd/container"] = []byte("lxc\n")
	m.Files[moduleVersion] = []byte("595.91.07\n")
	m.Packages = effecttest.Packages{"linux-image-generic": {Version: "7.0.0-42.42", Status: "ii"}}
	return m
}

func TestNvidiaContainerWithTheSwitchOffInstallsNothing(t *testing.T) {
	m := containerWithCard()
	units, _ := guestUnits(t, m, "---\n")
	for _, name := range nvidiaUnits {
		if got := applyUnit(t, units, name); got != nil {
			t.Errorf("%s planned %q, want nothing: the card's driver is no container's to install", name, got)
		}
	}
	for _, c := range m.Calls {
		if strings.HasPrefix(c, "run apt-get") || strings.HasPrefix(c, "root ") {
			t.Errorf("the container ran %q, want no install", c)
		}
	}
}

func TestNvidiaContainerWithTheSwitchOnInstallsTheUserspaceAtTheHostsModule(t *testing.T) {
	m := containerWithCard()
	units, out := guestUnits(t, m, "---\nWS_NVIDIA_GPU: true\n")
	steps := unitSteps(units)
	userspace := userspaceAt("595.91.07")
	if got, err := planned(t, steps, "packages/nvidia-userspace"); err != nil || !slices.Equal(got, []string{userspace}) {
		t.Errorf("userspace plan = %q, %v; want %q alone: no driver and no module", got, err, userspace)
	}
	for _, name := range []string{"packages/nvidia", "state/nvidia-reboot"} {
		if got, err := planned(t, steps, name); err != nil || got != nil {
			t.Errorf("%s planned %q, %v; want nothing in a container", name, got, err)
		}
	}
	for _, line := range out.Lines {
		if strings.HasPrefix(line, "warn: ") {
			t.Errorf("said %q, want no reboot note in a container", line)
		}
	}
	applyUnit(t, units, "file/nvidia-userspace-pin")
	want := "# Written by github.com/zadykian/ws, whose next run replaces any change made here.\n" +
		"Package: " + strings.Join(userspaceNames, " ") + "\nPin: version 595.91.07-*\nPin-Priority: 1001\n"
	if got := string(m.Files[nvidiaPin]); got != want {
		t.Errorf("the pin:\n%s\nwant:\n%s", got, want)
	}
	if got := applyUnit(t, units, "file/nvidia-userspace-pin"); got != nil {
		t.Errorf("a second run planned %q, want none", got)
	}
}

func TestNvidiaContainerFollowsTheHostsNewModule(t *testing.T) {
	m := containerWithCard()
	userspaceInstalled(m, "595.91.07")
	units, _ := guestUnits(t, m, "---\nWS_NVIDIA_GPU: true\n")
	if got, err := planned(t, unitSteps(units), "packages/nvidia-userspace"); err != nil || got != nil {
		t.Errorf("plan at the host's module = %q, %v; want none", got, err)
	}
	if got := applyUnit(t, units, "file/nvidia-userspace-pin"); !slices.Contains(got, "create "+nvidiaPin) {
		t.Fatalf("plan = %q, want the pin written", got)
	}
	m.Files[moduleVersion] = []byte("595.100.01\n")
	units, _ = guestUnits(t, m, "---\nWS_NVIDIA_GPU: true\n")
	steps := unitSteps(units)
	if got, err := planned(t, steps, "packages/nvidia-userspace"); err != nil || !slices.Equal(got, []string{userspaceAt("595.100.01")}) {
		t.Errorf("plan after the host's reboot = %q, %v; want the userspace pinned again", got, err)
	}
	if got := applyUnit(t, units, "file/nvidia-userspace-pin"); len(got) != 1 || !strings.Contains(string(m.Files[nvidiaPin]), "595.100.01-*") {
		t.Errorf("plan = %q, pin %q; want the pin moved", got, m.Files[nvidiaPin])
	}
}

func TestNvidiaContainerFailsWhereTheHostLoadsNoModule(t *testing.T) {
	m := containerWithCard()
	delete(m.Files, moduleVersion)
	units, _ := guestUnits(t, m, "---\nWS_NVIDIA_GPU: true\n")
	for _, u := range units {
		if u.Name != "packages/nvidia-userspace" && u.Name != "file/nvidia-userspace-pin" {
			continue
		}
		if _, err := u.Step.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "loads no nvidia module") {
			t.Errorf("%s plan = %v, want the missing module named", u.Name, err)
		}
	}
}

func TestNvidiaCloudVMInstallsTheDriverWhateverItsSwitch(t *testing.T) {
	for _, yml := range []string{"---\n", "---\nWS_NVIDIA_GPU: true\n"} {
		m := withNvidia()
		m.Packages = effecttest.Packages{"linux-image-generic": {Version: "7.0.0-42.42", Status: "ii"}}
		units, _ := guestUnits(t, m, yml)
		steps := unitSteps(units)
		want := "install " + nvidiaDriver + " " + nvidiaSMI + " " + nvidiaModules + "generic=7.0.0-42.42"
		if got, err := planned(t, steps, "packages/nvidia"); err != nil || !slices.Equal(got, []string{want}) {
			t.Errorf("%q: driver plan = %q, %v; want %q", yml, got, err, want)
		}
		for _, name := range []string{"packages/nvidia-userspace", "file/nvidia-userspace-pin"} {
			if got, err := planned(t, steps, name); err != nil || got != nil {
				t.Errorf("%q: %s planned %q, %v; want nothing off a container", yml, name, got, err)
			}
		}
	}
}

// unitSteps maps each unit of units by its name.
func unitSteps(units []kind.Unit) map[string]step.Step {
	steps := map[string]step.Step{}
	for _, u := range units {
		steps[u.Name] = u.Step
	}
	return steps
}
