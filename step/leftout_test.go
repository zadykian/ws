package step_test

import (
	"context"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/step"
)

func TestRunWithLeavesStepsOut(t *testing.T) {
	x := []step.Change{change("x")}
	var log []string
	steps := []step.Step{
		&fake{name: "a", todo: x, log: &log},
		&fake{name: "b", waitsOn: "a", todo: x, log: &log},
		&fake{name: "c", todo: x, log: &log},
		&fake{name: "d", log: &log},
	}
	var ev []step.Event
	o := step.Options{
		Observer: observerFunc(func(e step.Event) { ev = append(ev, e) }),
		Parent:   "p",
		LeftOut:  func(parent, name string) bool { return parent == "p" && (name == "a" || name == "d") },
	}
	changed, err := step.RunWith(context.Background(), steps, o)
	if err != nil || !changed {
		t.Fatalf("changed, err = %v, %v", changed, err)
	}
	wantLog := []string{"plan a", "plan c", "apply c", "plan c", "plan d"}
	if !slices.Equal(log, wantLog) {
		t.Errorf("log = %q, want %q", log, wantLog)
	}
	last := map[string]step.Event{}
	for _, e := range ev {
		last[e.Step] = e
	}
	if e := last["a"]; e.Kind != step.Done || !e.LeftOut || len(e.Changes) != 1 {
		t.Errorf("a ends %+v, want Done, LeftOut, with its change", e)
	}
	if e := last["b"]; e.Kind != step.Skipped || e.After != "a" || e.LeftOut {
		t.Errorf("b ends %+v, want Skipped after a", e)
	}
	if e := last["c"]; e.Kind != step.Done || e.LeftOut {
		t.Errorf("c ends %+v, want Done, applied", e)
	}
	if e := last["d"]; !e.LeftOut || len(e.Changes) != 0 {
		t.Errorf("d ends %+v, want LeftOut with no change", e)
	}
}

func TestRunWithLeaveOutPrintsLikeNone(t *testing.T) {
	var log []string
	out := &effecttest.Out{}
	steps := []step.Step{&fake{name: "a", todo: []step.Change{change("x")}, log: &log}}
	o := step.Options{Observer: step.Printer{Out: out}, LeftOut: func(string, string) bool { return true }}
	if _, err := step.RunWith(context.Background(), steps, o); err != nil {
		t.Fatal(err)
	}
	if got := out.Lines; len(got) != 0 {
		t.Errorf("printed %q, want nothing", got)
	}
}

func TestRunWithCheckLeavesNoneOut(t *testing.T) {
	var log []string
	steps := []step.Step{&fake{name: "a", todo: []step.Change{change("x")}, log: &log}}
	var ev []step.Event
	o := step.Options{Check: true, LeftOut: func(string, string) bool { return true },
		Observer: observerFunc(func(e step.Event) { ev = append(ev, e) })}
	if _, err := step.RunWith(context.Background(), steps, o); err != nil {
		t.Fatal(err)
	}
	for _, e := range ev {
		if e.LeftOut {
			t.Errorf("event %+v is LeftOut under check", e)
		}
	}
}

type observerFunc func(step.Event)

func (f observerFunc) Observe(e step.Event) { f(e) }
