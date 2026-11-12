package fans

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// applyService plans and applies the service step, and returns what its plan held.
func applyService(t *testing.T, m *effecttest.Machine, u *effecttest.Units) step.Plan {
	s := steps(values.Default().Fans, m, u, &effecttest.Out{}, exeAt)
	svc := s[len(s)-1]
	p, err := svc.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	return p
}

func serviceMachine() *effecttest.Machine {
	m := onTheBoard()
	m.Dirs["/run/systemd/system"] = 0o755
	m.Files["/usr/bin/ws"] = []byte("ws binary")
	return m
}

func TestServiceInstallsEnablesAndStartsThenChangesNothing(t *testing.T) {
	m, u := serviceMachine(), &effecttest.Units{}
	if p := applyService(t, m, u); len(p.Changes) < 4 {
		t.Fatalf("first plan = %d changes, want the three files, the reload and the unit", len(p.Changes))
	}
	if string(m.Files[binaryPath]) != "ws binary" || m.Files[configPath] == nil || m.Files[unitPath] == nil {
		t.Error("the binary, the config or the unit is not written")
	}
	if got := u.States["ws-fans.service"]; !got.Enabled || !got.Active {
		t.Errorf("ws-fans = %+v, want enabled and running", got)
	}
	if p := applyService(t, m, u); len(p.Changes) != 0 {
		t.Errorf("second plan = %d changes, want none", len(p.Changes))
	}
}

func TestServiceRestartsOnANewBinary(t *testing.T) {
	m, u := serviceMachine(), &effecttest.Units{}
	applyService(t, m, u)
	u.Calls = nil
	m.Files["/usr/bin/ws"] = []byte("new ws binary")
	applyService(t, m, u)
	if !slices.Contains(u.Calls, "restart ws-fans.service") || string(m.Files[binaryPath]) != "new ws binary" {
		t.Errorf("calls = %q, binary = %q; want a restart on the new copy", u.Calls, m.Files[binaryPath])
	}
}

func TestServiceSkipsItsUnitWithoutSystemd(t *testing.T) {
	m, u := serviceMachine(), &effecttest.Units{}
	delete(m.Dirs, "/run/systemd/system")
	applyService(t, m, u)
	if len(u.Calls) != 0 {
		t.Errorf("calls = %q, want none", u.Calls)
	}
}
