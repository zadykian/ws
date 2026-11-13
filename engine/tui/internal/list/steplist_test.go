package list

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/step"
)

// namedStep is a step that a test lists and never runs.
type namedStep string

func (n namedStep) Name() string                            { return string(n) }
func (n namedStep) WaitsOn() string                         { return "" }
func (n namedStep) Plan(context.Context) (step.Plan, error) { return step.Plan{}, nil }
func (n namedStep) Apply(context.Context, step.Plan) error  { return nil }

func testList() StepList {
	return NewStepList([]Group{{Name: "base", Steps: []string{"Packages", "Locale"}}, {Name: "ssh", Steps: []string{"Keys"}}})
}

func TestNewGroupNamesTheSteps(t *testing.T) {
	got := NewGroup("base", []step.Step{namedStep("Packages"), namedStep("Locale")})
	want := Group{Name: "base", Steps: []string{"Packages", "Locale"}, Waits: map[string]string{}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewGroup = %+v, want %+v", got, want)
	}
}

// waitingStep is a step that waits on another.
type waitingStep struct {
	namedStep
	awaited string
}

func (w waitingStep) WaitsOn() string { return w.awaited }

func TestNewGroupNamesWhatAStepWaitsOn(t *testing.T) {
	got := NewGroup("signoz", []step.Step{namedStep("up"), waitingStep{"ready", "up"}})
	if want := map[string]string{"ready": "up"}; !reflect.DeepEqual(got.Waits, want) {
		t.Errorf("Waits = %v, want %v", got.Waits, want)
	}
}

func TestStateAfter(t *testing.T) {
	changes := []step.Change{oneChange("a")}
	cases := []struct {
		event step.Event
		want  StepState
	}{
		{step.Event{Kind: step.Planned, Changes: changes}, StepOpen},
		{step.Event{Kind: step.Started, Changes: changes}, StepOpen},
		{step.Event{Kind: step.Done}, StepSame},
		{step.Event{Kind: step.Done, Changes: changes}, StepChanged},
		{step.Event{Kind: step.Done, Changes: changes, LeftOut: true}, StepLeftOut},
		{step.Event{Kind: step.Failed, Err: errors.New("no")}, StepFailed},
		{step.Event{Kind: step.Skipped, After: "Keys"}, StepSkipped},
		{step.Event{Kind: step.Skipped, After: "Keys", LeftOut: true}, StepLeftOut},
	}
	for _, c := range cases {
		if got := StateAfter(c.event); got != c.want {
			t.Errorf("stateAfter(%+v) = %d, want %d", c.event, got, c.want)
		}
	}
}

func TestStepRowLabelNamesItsParent(t *testing.T) {
	if got := (StepRow{Parent: "base", Name: "Packages"}).Label(); got != "base: Packages" {
		t.Errorf("label = %q", got)
	}
	if got := (StepRow{Name: "Packages"}).Label(); got != "Packages" {
		t.Errorf("label = %q", got)
	}
}

func TestStepListTellsARepeatedNameByItsPlace(t *testing.T) {
	list := NewStepList([]Group{{Steps: []string{"ssh", "Tools", "ssh"}}})
	list = list.Take(step.Event{Kind: step.Done, Step: "ssh", Index: 2})
	if list[0].State != StepOpen || list[2].State != StepSame {
		t.Errorf("states = %d, %d, want the last ssh alone ended", list[0].State, list[2].State)
	}
}

func TestStepListTakeLeavesAnOlderValueAlone(t *testing.T) {
	old := testList()
	_ = old.Take(step.Event{Kind: step.Done, Parent: "base", Step: "Packages"})
	if old[0].State != StepOpen {
		t.Error("take changed the list it was called on")
	}
}

func TestStepListIgnoresAStepItDoesNotHold(t *testing.T) {
	list := testList().Take(step.Event{Kind: step.Done, Parent: "other", Step: "Packages"})
	if list.Ended() != 0 {
		t.Errorf("ended = %d, want 0", list.Ended())
	}
}

func TestStepListPlannedOpensAnEndedStep(t *testing.T) {
	list := testList().Take(step.Event{Kind: step.Done, Parent: "base", Step: "Packages"})
	list = list.Take(step.Event{Kind: step.Planned, Parent: "base", Step: "Packages"})
	if list.Ended() != 0 || list.At() != 0 {
		t.Errorf("ended = %d, at = %d, want the step open again", list.Ended(), list.At())
	}
}

func TestStepListAt(t *testing.T) {
	list := testList()
	if list.At() != 0 {
		t.Errorf("at = %d before any event, want the first step", list.At())
	}
	list = list.Take(step.Event{Kind: step.Done, Parent: "base", Step: "Packages"})
	if list.At() != 1 {
		t.Errorf("at = %d past the first step, want the second", list.At())
	}
	failed := list.Take(step.Event{Kind: step.Failed, Parent: "base", Step: "Locale", Index: 1})
	failed = failed.Take(step.Event{Kind: step.Skipped, Parent: "ssh", Step: "Keys"})
	if failed.At() != 1 {
		t.Errorf("at = %d past a failure, want the failed step", failed.At())
	}
	list = list.Take(step.Event{Kind: step.Done, Parent: "base", Step: "Locale", Index: 1})
	list = list.Take(step.Event{Kind: step.Done, Parent: "ssh", Step: "Keys"})
	if list.At() != -1 || list.Ended() != 3 {
		t.Errorf("at = %d, ended = %d, want a run that is over", list.At(), list.Ended())
	}
}

func TestStepListViewMarksEachStateByItsShape(t *testing.T) {
	list := NewStepList([]Group{{Name: "base", Steps: []string{"a", "b", "c", "d", "e", "f", "g"}}})
	changes := []step.Change{oneChange("x")}
	events := []step.Event{
		{Kind: step.Done},
		{Kind: step.Done, Changes: changes},
		{Kind: step.Done, Changes: changes, LeftOut: true},
		{Kind: step.Skipped},
	}
	for i, e := range events {
		e.Parent, e.Index = "base", i
		list = list.Take(e)
	}
	want := "base\n  ✓ a\n  ● b\n  ○ c\n  – d\n  ▸ e\n  · f\n  · g"
	if got := stripped(list.View(testTheme, 20)); got != want {
		t.Errorf("view =\n%s\nwant\n%s", got, want)
	}
}

func TestStepListViewDrawsNoHeadingForAFlatList(t *testing.T) {
	got := stripped(NewStepList([]Group{{Steps: []string{"Tools", "ssh"}}}).View(testTheme, 20))
	if want := "▸ Tools\n· ssh"; got != want {
		t.Errorf("view =\n%s\nwant\n%s", got, want)
	}
}

func TestStepListViewKeepsTheCurrentStepInSight(t *testing.T) {
	names := make([]string, 40)
	for i := range names {
		names[i] = "step" + string(rune('A'+i))
	}
	list := NewStepList([]Group{{Name: "base", Steps: names}})
	for i := range 30 {
		list = list.Take(step.Event{Kind: step.Done, Parent: "base", Index: i})
	}
	got := strings.Split(list.View(testTheme, 9), "\n")
	if len(got) != 9 || !strings.Contains(got[3], "▸") {
		t.Errorf("view = %q, want 9 lines with the current step on the fourth", got)
	}
}

func TestStepListViewEndsOnTheLastStepOfAFinishedRun(t *testing.T) {
	list := NewStepList([]Group{{Steps: []string{"a", "b", "c", "d"}}})
	for i := range 4 {
		list = list.Take(step.Event{Kind: step.Done, Index: i})
	}
	if got, want := stripped(list.View(testTheme, 2)), "✓ c\n✓ d"; got != want {
		t.Errorf("view =\n%s\nwant\n%s", got, want)
	}
}
