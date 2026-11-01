package wizard

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/step"
)

type change string

func (c change) String() string { return string(c) }

// stage has one change to make until Apply.
type stage struct {
	done bool
}

func (*stage) Name() string    { return "Stage" }
func (*stage) WaitsOn() string { return "" }

func (s *stage) Plan(context.Context) (step.Plan, error) {
	if s.done {
		return step.Plan{}, nil
	}
	return step.Plan{Changes: []step.Change{change("would act")}}, nil
}

func (s *stage) Apply(context.Context, step.Plan) error {
	s.done = true
	return nil
}

func TestRunHoldsLockAndCaffeinate(t *testing.T) {
	m := effecttest.NewMachine()
	var out effecttest.Out
	changed, err := Run(context.Background(), m, &out, []step.Step{&stage{}}, false, "/tmp", 42, nil)
	if err != nil || !changed {
		t.Fatalf("Run = %v, %v; want true, nil", changed, err)
	}
	for _, call := range []string{"mkdir /tmp/ws-wizard.lock 700", "run caffeinate -i -w 42", "run rm -rf /tmp/ws-wizard.lock"} {
		if !slices.Contains(m.Calls, call) {
			t.Errorf("calls %v lack %q", m.Calls, call)
		}
	}
}

func TestRunStopsWhereLockIsHeld(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/tmp/ws-wizard.lock"] = 0o700
	if _, err := Run(context.Background(), m, &effecttest.Out{}, []step.Step{&stage{}}, false, "/tmp", 42, nil); err == nil {
		t.Fatal("Run succeeded with the lock held")
	}
	if slices.Contains(m.Calls, "run caffeinate -i -w 42") || slices.Contains(m.Calls, "run rm -rf /tmp/ws-wizard.lock") {
		t.Errorf("a run that took no lock acted or removed it: %v", m.Calls)
	}
}

func TestRunClosesUnderTheLock(t *testing.T) {
	m := effecttest.NewMachine()
	var held, closed bool
	closing := func(context.Context) {
		_, held = m.Dirs["/tmp/ws-wizard.lock"]
		closed = true
	}
	if _, err := Run(context.Background(), m, &effecttest.Out{}, []step.Step{&stage{}}, false, "/tmp", 42, closing); err != nil {
		t.Fatal(err)
	}
	if !closed || !held {
		t.Errorf("closing ran = %v, with the lock held = %v; want both", closed, held)
	}
}
