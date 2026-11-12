package ikeward

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func statusOf(m *effecttest.Machine, tries int) Status {
	m.Dirs["/run/systemd/system"] = 0o755
	return Status{M: m, Out: &effecttest.Out{}, Tries: tries, Wait: time.Millisecond}
}

func TestStatusPlansNothingWhereTheVPNIsUp(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[binary+" status"] = effecttest.Result{}
	if changes, err := statusOf(m, 3).Plan(t.Context()); err != nil || len(changes) != 0 {
		t.Errorf("plan = %v, %v; want nothing", changes, err)
	}
}

func TestStatusWaitsAndFailsAfterItsTries(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[binary+" status"] = effecttest.Result{Err: errors.New("down")}
	changes, err := statusOf(m, 3).Plan(t.Context())
	if err != nil || len(changes) != 1 {
		t.Fatalf("plan = %v, %v; want the wait", changes, err)
	}
	if err := changes[0].Apply(t.Context()); err == nil || !strings.Contains(err.Error(), "3 tries") {
		t.Errorf("apply = %v, want a failure after 3 tries", err)
	}
}

func TestStatusSaysThatSystemdIsNotRunning(t *testing.T) {
	s := statusOf(effecttest.NewMachine(), 3)
	delete(s.M.(*effecttest.Machine).Dirs, "/run/systemd/system")
	if changes, err := s.Plan(t.Context()); err != nil || len(changes) != 0 || len(s.Out.(*effecttest.Out).Lines) != 1 {
		t.Errorf("plan = %v, %v; want nothing and a line", changes, err)
	}
}
