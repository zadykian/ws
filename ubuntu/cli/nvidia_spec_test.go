package cli

import (
	"errors"
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// The names of nvidia's branch, as its templates give them.
const (
	nvidiaDriver  = "nvidia-headless-no-dkms-595-server-open"
	nvidiaSMI     = "nvidia-utils-595-server"
	nvidiaModules = "linux-modules-nvidia-595-server-open-"
)

// nvidiaFlavors are the kernel flavors that the archive builds the modules for.
var nvidiaFlavors = []string{"generic", "generic-7.0", "generic-hwe-26.04", "generic-hwe-26.04-edge"}

// nvidiaCheck is the simulated install of every name of the branch.
func nvidiaCheck() string {
	names := []string{nvidiaDriver, nvidiaSMI}
	for _, f := range nvidiaFlavors {
		names = append(names, nvidiaModules+f)
	}
	return "apt-get install -s " + strings.Join(names, " ")
}

// withNvidia is a machine with NVIDIA's display device.
func withNvidia() *effecttest.Machine {
	m := effecttest.NewMachine()
	withPCI(m, "0000:01:00.0", "0x030000", "0x10de")
	return m
}

func TestNvidiaOffWithoutTheGPU(t *testing.T) {
	virtio, audio := effecttest.NewMachine(), effecttest.NewMachine()
	withPCI(virtio, "0000:00:01.0", "0x030000", "0x1af4")
	withPCI(audio, "0000:01:00.1", "0x040300", "0x10de")
	for name, m := range map[string]*effecttest.Machine{"virtio's display": virtio, "NVIDIA's audio": audio,
		"no PCI devices": effecttest.NewMachine()} {
		out := &effecttest.Out{}
		m.Results[nvidiaCheck()] = effecttest.Result{}
		steps := specUnits(t, m, out)
		for _, unit := range []string{"packages/nvidia-archive", "packages/nvidia", "state/nvidia-reboot"} {
			if got, err := planned(t, steps, unit); err != nil || got != nil {
				t.Errorf("%s: %s plans %q, %v; want none", name, unit, got, err)
			}
		}
		want := []string{`say: packages/nvidia: off on this machine, as its gpu_vendor is not "0x10de"`,
			`say: state/nvidia-reboot: off on this machine, as its gpu_vendor is not "0x10de"`}
		if !slices.Equal(out.Lines, want) {
			t.Errorf("%s: said %q, want %q", name, out.Lines, want)
		}
		if runs := slices.DeleteFunc(slices.Clone(m.Calls), func(c string) bool { return !strings.HasPrefix(c, "run ") }); !slices.Equal(runs, []string{"run " + nvidiaCheck()}) {
			t.Errorf("%s: ran %q, want the simulation alone", name, runs)
		}
	}
}

func TestNvidiaArchiveOffWithTheGPU(t *testing.T) {
	m, out := withNvidia(), &effecttest.Out{}
	m.Results[nvidiaCheck()] = effecttest.Result{Err: errors.New("E: Unable to locate package " + nvidiaSMI)}
	if got, err := planned(t, specUnits(t, m, out), "packages/nvidia-archive"); err != nil || got != nil {
		t.Errorf("plan = %q, %v; want none", got, err)
	}
	want := []string{`say: packages/nvidia-archive: off on this machine, as its gpu_vendor is "0x10de"`}
	if !slices.Equal(out.Lines, want) || slices.Contains(m.Calls, "run "+nvidiaCheck()) {
		t.Errorf("said %q, ran %q; want %q and no simulation", out.Lines, m.Calls, want)
	}
}

func TestNvidiaArchiveChecksEveryName(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[nvidiaCheck()] = effecttest.Result{Err: errors.New("E: Unable to locate package " + nvidiaSMI)}
	if _, err := planned(t, specUnits(t, m, &effecttest.Out{}), "packages/nvidia-archive"); err == nil ||
		!strings.Contains(err.Error(), "Unable to locate") {
		t.Errorf("plan with a name gone = %v; want apt's error", err)
	}
}

func TestNvidiaInstallsTheFlavorsModules(t *testing.T) {
	for _, tc := range []struct {
		name     string
		packages effecttest.Packages
		modules  []string
	}{
		{"generic", effecttest.Packages{"linux-image-generic": {Version: "7.0.0-42.42", Status: "ii"},
			nvidiaModules + "generic": {Version: "7.0.0-14.14", Status: "ii"}},
			[]string{"generic=7.0.0-42.42"}},
		{"hwe and a removed generic", effecttest.Packages{"linux-image-generic": {Version: "7.0.0-14.14", Status: "rc"},
			"linux-image-generic-hwe-26.04": {Version: "7.0.0-43.43", Status: "ii"}},
			[]string{"generic-hwe-26.04=7.0.0-43.43"}},
		{"two", effecttest.Packages{"linux-image-generic": {Version: "7.0.0-42.42", Status: "ii"},
			"linux-image-generic-7.0": {Version: "7.0.0-43.43", Status: "ii"}},
			[]string{"generic=7.0.0-42.42", "generic-7.0=7.0.0-43.43"}},
	} {
		for _, class := range []string{"0x030000", "0x030200"} {
			m := effecttest.NewMachine()
			withPCI(m, "0000:01:00.0", class, "0x10de")
			m.Packages = tc.packages
			want := "install " + nvidiaDriver + " " + nvidiaSMI
			for _, mod := range tc.modules {
				want += " " + nvidiaModules + mod
			}
			if got, err := planned(t, specUnits(t, m, &effecttest.Out{}), "packages/nvidia"); err != nil || !slices.Equal(got, []string{want}) {
				t.Errorf("%s, class %s: plan = %q, %v; want %q", tc.name, class, got, err, want)
			}
		}
	}
}

func TestNvidiaRefusesAnotherKernel(t *testing.T) {
	m := withNvidia()
	m.Packages = effecttest.Packages{"linux-image-generic-hwe-24.04": {Version: "6.8.0-90.91", Status: "ii"}}
	if _, err := planned(t, specUnits(t, m, &effecttest.Out{}), "packages/nvidia"); err == nil ||
		!strings.Contains(err.Error(), "no NVIDIA modules") {
		t.Errorf("plan = %v; want the flavors refused", err)
	}
}

func TestNvidiaRebootNote(t *testing.T) {
	for loaded, want := range map[string]string{"": "nvidia at none", "595.91.07\n": ""} {
		m, out := withNvidia(), &effecttest.Out{}
		m.Packages = effecttest.Packages{nvidiaSMI: {Version: "595.91.07-0ubuntu0.26.04.2", Status: "ii"}}
		if loaded != "" {
			m.Files["/sys/module/nvidia/version"] = []byte(loaded)
		}
		got, err := planned(t, specUnits(t, m, out), "state/nvidia-reboot")
		said := strings.Join(out.Lines, "\n")
		if err != nil || got != nil {
			t.Errorf("loaded %q: plan = %q, %v; want none", loaded, got, err)
		}
		if want == "" && said != "" || want != "" && (len(out.Lines) != 1 || !strings.HasPrefix(said, "warn: ") ||
			!strings.Contains(said, want) || !strings.Contains(said, "ws-fans reads no GPU")) {
			t.Errorf("loaded %q: said %q, want %q", loaded, said, want)
		}
	}
}

// TestNvidiaNamesOneBranch holds the three places that name the branch together: the packages'
// template, the archive's, and the reboot's package.
func TestNvidiaNamesOneBranch(t *testing.T) {
	m := withNvidia()
	m.Packages = effecttest.Packages{}
	for _, f := range nvidiaFlavors {
		m.Packages["linux-image-"+f] = effecttest.Package{Version: "7.0.0-42.42", Status: "ii"}
	}
	steps := specUnits(t, m, &effecttest.Out{})
	got, err := planned(t, steps, "packages/nvidia")
	if err != nil || len(got) != 1 {
		t.Fatalf("plan = %q, %v; want one install", got, err)
	}
	var names []string
	for _, n := range strings.Fields(strings.TrimPrefix(got[0], "install ")) {
		name, _, _ := strings.Cut(n, "=")
		names = append(names, name)
	}
	// The check runs where the GPU is not.
	const simulation = "run apt-get install -s "
	bare := effecttest.NewMachine()
	_, _ = planned(t, specUnits(t, bare, &effecttest.Out{}), "packages/nvidia-archive")
	i := slices.IndexFunc(bare.Calls, func(c string) bool { return strings.HasPrefix(c, simulation) })
	if i < 0 {
		t.Fatalf("the archive's check ran %q, no simulation", bare.Calls)
	}
	if archive := strings.Fields(strings.TrimPrefix(bare.Calls[i], simulation)); !slices.Equal(names, archive) {
		t.Errorf("the packages' names %q, the archive's %q", names, archive)
	}
	// The spec is a template, so a regexp reads the reboot's package from its text.
	spec, err := os.ReadFile(rolesDir + "/nvidia/units.toml.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	found := regexp.MustCompile(`(?m)^package = "([^"]+)"`).FindAllSubmatch(spec, -1)
	if len(found) != 1 {
		t.Fatalf("the spec names %d packages for the reboot, want one", len(found))
	}
	if p := string(found[0][1]); !slices.Contains(names, p) {
		t.Errorf("the reboot's package %q is none of %q", p, names)
	}
}
