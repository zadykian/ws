package step_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/zadykian/ws/engine/core/step"
)

func TestReadShowsAPlan(t *testing.T) {
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

// titled is a change whose text names nothing, as a bare diff.
type titled string

func (t titled) String() string { return string(t) }
func (titled) Title() string    { return "would write /a" }

func TestPendingOfPutsATitleFirst(t *testing.T) {
	if got, want := step.PendingOf(titled("+a\n")).Text, "would write /a\n+a\n"; got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
	if got := step.PendingOf(change("b")).Text; got != "b" {
		t.Errorf("text = %q, want the change's own", got)
	}
}

// quoting is a change whose String quotes a file of the user's.
type quoting struct{ titled }

func (quoting) Excerpt() string { return "+ws\n" }

func TestPendingOfHoldsAnExcerpt(t *testing.T) {
	if got, want := step.PendingOf(quoting{"+ws\n the user's text\n"}).Text, "would write /a\n+ws\n"; got != want {
		t.Errorf("text = %q, want %q", got, want)
	}
	if got, want := step.Headline(quoting{"+ws\n"}), "would write /a"; got != want {
		t.Errorf("headline = %q, want %q", got, want)
	}
}

func TestReadAppliesNothing(t *testing.T) {
	log := []string{}
	if _, err := step.Read(t.Context(), []step.Step{&fake{name: "a", todo: []step.Change{change("x")}, log: &log}}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"plan a"}; !reflect.DeepEqual(log, want) {
		t.Errorf("log = %v, want the plan alone", log)
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

func TestReadGoesPastAFailure(t *testing.T) {
	log := []string{}
	boom := errors.New("two routes")
	got, err := step.Read(t.Context(), []step.Step{
		&fake{name: "a", planErr: boom, log: &log},
		&fake{name: "b", waitsOn: "a", log: &log},
		&fake{name: "c", todo: []step.Change{change("x")}, log: &log},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []step.Report{
		{Step: "a", Err: boom},
		{Step: "b", After: "a"},
		{Step: "c", Changes: []step.Pending{{Kind: step.OtherChange, Text: "x"}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("reports = %+v, want %+v", got, want)
	}
}

func TestReadNamesAStepWaitingOnNoEarlierOne(t *testing.T) {
	log := []string{}
	got, err := step.Read(t.Context(), []step.Step{&fake{name: "a", waitsOn: "z", log: &log}})
	if err != nil {
		t.Fatal(err)
	}
	if want := `a waits on "z", which is not an earlier step`; got[0].Err == nil || got[0].Err.Error() != want {
		t.Errorf("err = %v, want %q", got[0].Err, want)
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
