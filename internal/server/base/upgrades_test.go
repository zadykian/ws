package base

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
)

// upgrades is the step named Upgrades that Steps builds on m and u.
func upgrades(t *testing.T, m effect.Machine, u effect.Units, out effect.Out) step.Step {
	for _, s := range Steps(values.Default(), secrets.Secrets{}, m, u, out) {
		if s.Name() == "Upgrades" {
			return s
		}
	}
	t.Fatal("Steps has no step named Upgrades")
	return nil
}

func TestUpgrades(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	u := &effecttest.Units{}
	out := &effecttest.Out{}
	if changed, err := step.Run(t.Context(), []step.Step{upgrades(t, m, u, out)}, out, false); err != nil || !changed {
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
	if _, err := step.Run(t.Context(), []step.Step{upgrades(t, m, &effecttest.Units{}, out)}, out, false); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Files["/etc/apt/apt.conf.d/20auto-upgrades"]; !ok {
		t.Error("the file is not written")
	}
	if !strings.Contains(strings.Join(out.Lines, "\n"), "skipped, systemd is not running") {
		t.Errorf("no skip said: %v", out.Lines)
	}
}
