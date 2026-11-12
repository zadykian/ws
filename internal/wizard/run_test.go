package wizard

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
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
	changed, err := Run(context.Background(), m, step.Options{Observer: step.Printer{Out: &out}}, []step.Step{&stage{}}, "/tmp", 42, nil)
	if err != nil || !changed {
		t.Fatalf("Run = %v, %v; want true, nil", changed, err)
	}
	for _, call := range []string{"mkdir /tmp/ws-wizard.lock 700", "run caffeinate -i -w 42", "run rm -rf /tmp/ws-wizard.lock"} {
		if !slices.Contains(m.Calls, call) {
			t.Errorf("calls %v lack %q", m.Calls, call)
		}
	}
	if want := []string{"say: Stage: applying 1 change(s)"}; !slices.Equal(out.Lines, want) {
		t.Errorf("lines = %q, want %q", out.Lines, want)
	}
}

// On ws, with no pid, the run holds its lock and runs no caffeinate.
func TestRunOnWsKeepsNothingAwake(t *testing.T) {
	m := effecttest.NewMachine()
	if _, err := Run(context.Background(), m, step.Options{}, []step.Step{&stage{}}, "/tmp", 0, nil); err != nil {
		t.Fatal(err)
	}
	if want := []string{"mkdir /tmp/ws-wizard.lock 700", "readdir /tmp/ws-wizard.lock", "run rm -rf /tmp/ws-wizard.lock"}; !slices.Equal(m.Calls, want) {
		t.Errorf("calls = %q, want %q", m.Calls, want)
	}
}

func TestRunStopsWhereLockIsHeld(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/tmp/ws-wizard.lock"] = 0o700
	if _, err := Run(context.Background(), m, step.Options{}, []step.Step{&stage{}}, "/tmp", 42, nil); err == nil {
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
	if _, err := Run(context.Background(), m, step.Options{}, []step.Step{&stage{}}, "/tmp", 42, closing); err != nil {
		t.Fatal(err)
	}
	if !closed || !held {
		t.Errorf("closing ran = %v, with the lock held = %v; want both", closed, held)
	}
}
