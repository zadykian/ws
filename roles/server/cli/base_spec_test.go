package cli

import (
	"bytes"
	"os"
	"slices"
	"strings"
	"testing"
	"text/template"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/roles/server/values"
)

// The swap and vm.swappiness are the kernel's, which a container shares with its host, so base's
// units that set them are tried on the fake machine alone.

const (
	swappinessFile = "/etc/sysctl.d/99-swappiness.conf"
	// inForce and loads are the commands of service/base-sysctl's drift, as the fake keys them.
	inForce = "cat /proc/sys/vm/swappiness"
	loads   = `awk -F  *= * $1 == "vm.swappiness" { print $2 } ` + swappinessFile
)

// baseUnits loads the specs for the default values on m and u, and returns their units in the
// order they run, with what they say on out.
func baseUnits(t *testing.T, m *effecttest.Machine, u *effecttest.Units) ([]kind.Unit, *effecttest.Out) {
	v, err := values.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	out := &effecttest.Out{}
	return loadSpecs(t, kind.Env{M: m, U: u, Out: out, Data: v, Self: selfPath}), out
}

func TestBaseUpgradesFileIsTodays(t *testing.T) {
	want, err := os.ReadFile("../ubuntu/base/files/20auto-upgrades")
	if err != nil {
		t.Fatal(err)
	}
	m, u := effecttest.NewMachine(), &effecttest.Units{}
	units, _ := baseUnits(t, m, u)
	applyUnit(t, units, "file/base-upgrades")
	const path = "/etc/apt/apt.conf.d/20auto-upgrades"
	if got := m.Files[path]; !bytes.Equal(got, want) || m.Modes[path] != 0o644 || m.Owners[path] != [2]int{} {
		t.Errorf("the file:\n%s\nmode %o, owner %v; want today's, root's at 644", got, m.Modes[path], m.Owners[path])
	}
	timers := []string{"apt-daily.timer", "apt-daily-upgrade.timer", "unattended-upgrades.service"}
	for i, name := range []string{"service/base-apt-daily", "service/base-apt-daily-upgrade", "service/base-unattended-upgrades"} {
		if got := applyUnit(t, units, name); !slices.Equal(got, []string{"enable " + timers[i], "start " + timers[i]}) {
			t.Errorf("%s: plan %q, want it enabled and started", name, got)
		}
		if got := applyUnit(t, units, name); got != nil {
			t.Errorf("%s: a second plan %q, want none", name, got)
		}
	}
}

func TestBaseSwapOffInAContainer(t *testing.T) {
	for _, mark := range []string{"/.dockerenv", "/run/.containerenv", "/run/systemd/container"} {
		m := effecttest.NewMachine()
		m.Files[mark] = nil
		units, out := baseUnits(t, m, &effecttest.Units{})
		for _, name := range []string{"state/base-swap", "service/base-sysctl"} {
			if got := applyUnit(t, units, name); got != nil {
				t.Errorf("%s, %s: plan %q, want none", mark, name, got)
			}
			if !slices.Contains(out.Lines, "say: "+name+`: off on this machine, as its container is "yes"`) {
				t.Errorf("%s, %s: said %q, want the unit off", mark, name, out.Lines)
			}
		}
		if slices.ContainsFunc(m.Calls, func(c string) bool {
			return strings.Contains(c, "/proc/swaps") || strings.Contains(c, "/proc/sys/vm") || strings.HasPrefix(c, "root ")
		}) {
			t.Errorf("%s: calls %q, want no read of the swap or the kernel's settings", mark, m.Calls)
		}
	}
}

// The swap's plan reads the disk of the machine that runs it, never the fake's. So the test reads
// the spec as the values fill it, and state.swap's own test shows that size_mib makes the size.
func TestBaseSwapTakesTheValuesSize(t *testing.T) {
	text, err := os.ReadFile("../ubuntu/base/units.toml.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	spec := template.Must(template.New("units.toml.tmpl").Parse(string(text)))
	for yml, want := range map[string]string{
		"WS_BASE_SWAP_SIZE_MB: 64\n": "swap = { size_mib = 64, hint = ",
		"WS_BASE_SWAP_SIZE_MB: 0\n":  "swap = { size_mib = 0, hint = ",
		"":                           "swap = { hint = ",
	} {
		v, err := values.Parse([]byte(yml))
		if err != nil {
			t.Fatal(err)
		}
		var b strings.Builder
		if err := spec.Execute(&b, v); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(b.String(), "\n"+want) {
			t.Errorf("%q: the spec gives no line %q", yml, want)
		}
	}
}

func TestBaseSysctlRestartsOnDrift(t *testing.T) {
	for _, tc := range []struct {
		now     string
		restart bool
	}{{"60\n", true}, {"10\n", false}} {
		m, u := effecttest.NewMachine(), &effecttest.Units{States: map[string]effect.UnitState{
			"systemd-sysctl.service": {Active: true},
		}}
		m.Results[inForce] = effecttest.Result{Out: []byte(tc.now)}
		m.Results[loads] = effecttest.Result{Out: []byte("10\n")}
		units, _ := baseUnits(t, m, u)
		applyUnit(t, units, "file/base-swappiness")
		applyUnit(t, units, "service/base-sysctl")
		restarts := 0
		for _, c := range u.Calls {
			if c == "restart systemd-sysctl.service" {
				restarts++
			}
		}
		if want := map[bool]int{true: 1, false: 0}[tc.restart]; restarts != want || len(u.Calls) != want {
			t.Errorf("in force %q: calls %q, want %d restart and nothing else", tc.now, u.Calls, want)
		}
	}
}

func TestBaseSwappinessFileInAContainer(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/.dockerenv"] = nil
	units, _ := baseUnits(t, m, &effecttest.Units{})
	applyUnit(t, units, "file/base-swappiness")
	got := string(m.Files[swappinessFile])
	if !strings.HasPrefix(got, "# Written by github.com/zadykian/ws") || !strings.HasSuffix(got, "\nvm.swappiness = 10\n") {
		t.Errorf("the file:\n%s\nwant the header and vm.swappiness = 10", got)
	}
}
