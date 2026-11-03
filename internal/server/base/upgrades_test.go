package base

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/step"
)

func TestUpgrades(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	u := &effecttest.Units{}
	g := host.Group{Title: "Upgrades", Parts: upgradesParts(m, u, &effecttest.Out{})}
	if changed, err := step.Run(t.Context(), []step.Step{g}, &effecttest.Out{}, false); err != nil || !changed {
		t.Fatalf("changed = %v, err = %v", changed, err)
	}
	got := string(m.Files["/etc/apt/apt.conf.d/20auto-upgrades"])
	if !strings.Contains(got, `APT::Periodic::Unattended-Upgrade "1";`) {
		t.Errorf("20auto-upgrades = %q", got)
	}
	for _, name := range []string{"apt-daily.timer", "apt-daily-upgrade.timer", "unattended-upgrades.service"} {
		if s, _ := u.State(t.Context(), name); s != (effect.UnitState{Enabled: true, Active: true}) {
			t.Errorf("%s = %+v, want enabled and active", name, s)
		}
	}
}

func TestUpgradesSkipUnitsWithoutSystemd(t *testing.T) {
	m := effecttest.NewMachine()
	out := &effecttest.Out{}
	g := host.Group{Title: "Upgrades", Parts: upgradesParts(m, &effecttest.Units{}, out)}
	if _, err := step.Run(t.Context(), []step.Step{g}, out, false); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Files["/etc/apt/apt.conf.d/20auto-upgrades"]; !ok {
		t.Error("the file is not written")
	}
	if !strings.Contains(strings.Join(out.Lines, "\n"), "skipped, systemd is not running") {
		t.Errorf("no skip said: %v", out.Lines)
	}
}
