package step_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/zadykian/ws/step"
)

func TestReadShowsAPlanAloneWithoutAReporter(t *testing.T) {
	log := []string{}
	s := &fake{name: "a", todo: []step.Change{removal{"remove /a"}, change("b")}, log: &log}
	got, err := step.Read(t.Context(), []step.Step{s})
	if err != nil {
		t.Fatal(err)
	}
	want := []step.Report{{Step: "a", Changes: []step.Pending{
		{Kind: step.Removal, Text: "remove /a"}, {Kind: step.OtherChange, Text: "b"}}}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("reports = %+v, want %+v", got, want)
	}
}

func TestReadSkipsAStepWhoseAwaitedStepHasChanges(t *testing.T) {
	log := []string{}
	got, err := step.Read(t.Context(), []step.Step{
		&fake{name: "a", todo: []step.Change{change("x")}, log: &log},
		&fake{name: "b", waitsOn: "a", log: &log},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(log, []string{"plan a"}) {
		t.Errorf("log = %v, want b not planned", log)
	}
	if got[1].After != "a" {
		t.Errorf("after = %q, want the awaited step", got[1].After)
	}
}

func TestReadEndsWithItsContext(t *testing.T) {
	log := []string{}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	got, err := step.Read(ctx, []step.Step{&fake{name: "a", log: &log}})
	if !errors.Is(err, context.Canceled) || len(got) != 0 || len(log) != 0 {
		t.Errorf("reports = %+v, log = %v, err = %v, want nothing read and ctx's error", got, log, err)
	}
}
