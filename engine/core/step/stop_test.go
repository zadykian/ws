package step_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/step"
)

func TestRunWithStopEndsTheRunBetweenSteps(t *testing.T) {
	halted := errors.New("halted")
	var log []string
	j := &journal{}
	stop := func() error {
		if len(log) > 0 {
			return halted
		}
		return nil
	}
	steps := []step.Step{
		&fake{name: "a", todo: []step.Change{change("x")}, log: &log},
		&fake{name: "b", log: &log},
		&fake{name: "c", log: &log},
	}
	changed, err := step.RunWith(t.Context(), steps, step.Options{Observer: j, Stop: stop})
	if !errors.Is(err, halted) || !changed {
		t.Errorf("RunWith: changed %v, error %v", changed, err)
	}
	if want := []string{"plan a", "apply a", "plan a"}; !slices.Equal(log, want) {
		t.Errorf("log = %v, want %v", log, want)
	}
	want := []string{"planned /a#0 1", "started /a#0 1", "done /a#0 1", "skipped /b#1 0", "skipped /c#2 0"}
	if !slices.Equal(j.lines, want) {
		t.Errorf("events = %v, want %v", j.lines, want)
	}
}

func TestRunWithStopBeforeTheFirstStepRunsNone(t *testing.T) {
	halted := errors.New("halted")
	var log []string
	steps := []step.Step{&fake{name: "a", log: &log}}
	_, err := step.RunWith(t.Context(), steps, step.Options{Stop: func() error { return halted }})
	if !errors.Is(err, halted) || len(log) != 0 {
		t.Errorf("RunWith: %v, ran %v", err, log)
	}
}
