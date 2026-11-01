package host_test

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func unitFixture(booted bool) (*effecttest.Machine, *effecttest.Units, *effecttest.Out) {
	m := effecttest.NewMachine()
	if booted {
		m.Dirs["/run/systemd/system"] = 0o755
	}
	return m, &effecttest.Units{}, &effecttest.Out{}
}

func TestUnitEnablesAndStarts(t *testing.T) {
	m, u, out := unitFixture(true)
	unit := host.Unit{M: m, U: u, Out: out, Name: "a.service", Enabled: true, Run: host.Running}
	if got := applyAll(t, unit); !slices.Equal(got, []string{"enable a.service", "start a.service"}) {
		t.Errorf("first plan = %q, want enable and start", got)
	}
	if got := applyAll(t, unit); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}
