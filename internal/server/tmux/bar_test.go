package tmux

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/step"
)

// applyBar plans and applies the bar's step, and returns what its plan held.
func applyBar(t *testing.T, m *effecttest.Machine, u *effecttest.Units) step.Plan {
	s := steps(m, u, &effecttest.Out{}, func() (string, error) { return "/usr/bin/ws", nil })
	bar := s[len(s)-1]
	p, err := bar.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := bar.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	return p
}

func barMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	m.Files["/usr/bin/ws"] = []byte("ws binary")
	return m
}

func TestBarInstallsEnablesAndStartsThenChangesNothing(t *testing.T) {
	m, u := barMachine(), &effecttest.Units{}
	if p := applyBar(t, m, u); len(p.Changes) < 3 {
		t.Fatalf("first plan = %d changes, want the two files, the reload and the unit", len(p.Changes))
	}
	if string(m.Files[barBinary]) != "ws binary" || m.Files[barUnit] == nil {
		t.Error("the binary or the unit is not written")
	}
	if got := u.States["ws-bar.service"]; !got.Enabled || !got.Active {
		t.Errorf("ws-bar = %+v, want enabled and running", got)
	}
	if p := applyBar(t, m, u); len(p.Changes) != 0 {
		t.Errorf("second plan = %d changes, want none", len(p.Changes))
	}
	u.Calls = nil
	m.Files["/usr/bin/ws"] = []byte("new ws binary")
	applyBar(t, m, u)
	if !slices.Contains(u.Calls, "restart ws-bar.service") {
		t.Errorf("calls = %q, want a restart on the new copy", u.Calls)
	}
}

func TestBarSkipsItsUnitWithoutSystemd(t *testing.T) {
	m, u := barMachine(), &effecttest.Units{}
	delete(m.Dirs, "/run/systemd/system")
	if p := applyBar(t, m, u); len(p.Changes) != 0 || len(u.Calls) != 0 {
		t.Errorf("changes = %d, calls = %q; want none", len(p.Changes), u.Calls)
	}
}
