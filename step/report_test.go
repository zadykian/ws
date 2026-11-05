package step_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/zadykian/ws/step"
)

// reporting is a fake that says what it manages.
type reporting struct {
	*fake
	items []step.Item
	err   error
}

func (r reporting) Report(context.Context) ([]step.Item, error) {
	*r.log = append(*r.log, "report "+r.name)
	return r.items, r.err
}

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

func TestReadTakesTheItemsOfAReporter(t *testing.T) {
	log := []string{}
	items := []step.Item{{Kind: step.FileItem, Name: "/a", Want: "present", Found: "missing"}}
	got, err := step.Read(t.Context(), []step.Step{reporting{&fake{name: "a", log: &log}, items, nil}})
	if err != nil {
		t.Fatal(err)
	}
	if want := []step.Report{{Step: "a", Items: items}}; !reflect.DeepEqual(got, want) {
		t.Errorf("reports = %+v, want %+v", got, want)
	}
}

func TestReadSkipsAStepWhoseAwaitedStepHasChanges(t *testing.T) {
	log := []string{}
	got, err := step.Read(t.Context(), []step.Step{
		&fake{name: "a", todo: []step.Change{change("x")}, log: &log},
		reporting{fake: &fake{name: "b", waitsOn: "a", log: &log}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(log, []string{"plan a"}) {
		t.Errorf("log = %v, want b neither planned nor reported", log)
	}
	if got[1].After != "a" {
		t.Errorf("after = %q, want the awaited step", got[1].After)
	}
}

func TestReadGoesPastAFailure(t *testing.T) {
	log := []string{}
	got, err := step.Read(t.Context(), []step.Step{
		&fake{name: "a", planErr: errors.New("two routes"), log: &log},
		&fake{name: "b", waitsOn: "a", log: &log},
		&fake{name: "c", todo: []step.Change{change("x")}, log: &log},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []step.Report{
		{Step: "a", Items: []step.Item{{Kind: step.CheckItem, Name: "a", Want: "passes", Found: "fails",
			Conflict: "a: planning: two routes"}}},
		{Step: "b", After: "a"},
		{Step: "c", Changes: []step.Pending{{Kind: step.OtherChange, Text: "x"}}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("reports = %+v, want %+v", got, want)
	}
}

func TestReadShowsOneFailureOfAStep(t *testing.T) {
	log := []string{}
	found := []step.Item{{Kind: step.FileItem, Name: "/a"}}
	both := reporting{&fake{name: "a", planErr: errors.New("plan"), log: &log}, found, errors.New("report")}
	alone := reporting{&fake{name: "b", log: &log}, nil, errors.New("report")}
	got, err := step.Read(t.Context(), []step.Step{both, alone})
	if err != nil {
		t.Fatal(err)
	}
	if len(got[0].Items) != 1 || got[0].Items[0].Conflict != "a: planning: plan" {
		t.Errorf("items = %+v, want the plan's failure alone", got[0].Items)
	}
	if want := []string{"plan a", "plan b", "report b"}; !reflect.DeepEqual(log, want) {
		t.Errorf("log = %v, want %v: a failed plan keeps its step from being reported", log, want)
	}
	if len(got[1].Items) != 1 || got[1].Items[0].Conflict != "report" {
		t.Errorf("items = %+v, want the report's failure", got[1].Items)
	}
}

// itemized is a change that stands as an item.
type itemized string

func (i itemized) String() string { return "write " + string(i) }
func (i itemized) Item() step.Item {
	return step.Item{Kind: step.FileItem, Name: string(i), Found: "planned"}
}

func TestReadShowsTheItemsOfChangesNoReportNamed(t *testing.T) {
	log := []string{}
	named := step.Item{Kind: step.FileItem, Name: "/a", Found: "reported"}
	s := reporting{&fake{name: "s", todo: []step.Change{itemized("/a"), itemized("/b"), change("c")}, log: &log},
		[]step.Item{named}, nil}
	got, err := step.Read(t.Context(), []step.Step{s})
	if err != nil {
		t.Fatal(err)
	}
	want := []step.Item{named, itemized("/b").Item()}
	if !reflect.DeepEqual(got[0].Items, want) {
		t.Errorf("items = %+v, want %+v", got[0].Items, want)
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
