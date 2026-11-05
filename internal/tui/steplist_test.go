package tui

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/zadykian/ws/step"
)

// namedStep is a step that a test lists and never runs.
type namedStep string

func (n namedStep) Name() string                            { return string(n) }
func (n namedStep) WaitsOn() string                         { return "" }
func (n namedStep) Plan(context.Context) (step.Plan, error) { return step.Plan{}, nil }
func (n namedStep) Apply(context.Context, step.Plan) error  { return nil }

// oneChange is a plan's change, for the tests that read how many there are.
type oneChange string

func (c oneChange) String() string { return string(c) }

func testList() stepList {
	return newStepList([]Group{{"base", []string{"Packages", "Locale"}}, {"ssh", []string{"Keys"}}})
}

func TestNewGroupNamesTheSteps(t *testing.T) {
	got := NewGroup("base", []step.Step{namedStep("Packages"), namedStep("Locale")})
	want := Group{"base", []string{"Packages", "Locale"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NewGroup = %+v, want %+v", got, want)
	}
}

func TestStateAfter(t *testing.T) {
	changes := []step.Change{oneChange("a")}
	cases := []struct {
		event step.Event
		want  stepState
	}{
		{step.Event{Kind: step.Planned, Changes: changes}, stepOpen},
		{step.Event{Kind: step.Started, Changes: changes}, stepOpen},
		{step.Event{Kind: step.Done}, stepSame},
		{step.Event{Kind: step.Done, Changes: changes}, stepChanged},
		{step.Event{Kind: step.Done, Changes: changes, LeftOut: true}, stepLeftOut},
		{step.Event{Kind: step.Failed, Err: errors.New("no")}, stepFailed},
		{step.Event{Kind: step.Skipped, After: "Keys"}, stepSkipped},
	}
	for _, c := range cases {
		if got := stateAfter(c.event); got != c.want {
			t.Errorf("stateAfter(%+v) = %d, want %d", c.event, got, c.want)
		}
	}
}

func TestStepRowLabelNamesItsParent(t *testing.T) {
	if got := (stepRow{parent: "base", name: "Packages"}).label(); got != "base: Packages" {
		t.Errorf("label = %q", got)
	}
	if got := (stepRow{name: "Packages"}).label(); got != "Packages" {
		t.Errorf("label = %q", got)
	}
}

func TestStepListTellsARepeatedNameByItsPlace(t *testing.T) {
	list := newStepList([]Group{{"", []string{"ssh", "Tools", "ssh"}}})
	list = list.take(step.Event{Kind: step.Done, Step: "ssh", Index: 2})
	if list[0].state != stepOpen || list[2].state != stepSame {
		t.Errorf("states = %d, %d, want the last ssh alone ended", list[0].state, list[2].state)
	}
}

func TestStepListTakeLeavesAnOlderValueAlone(t *testing.T) {
	old := testList()
	_ = old.take(step.Event{Kind: step.Done, Parent: "base", Step: "Packages"})
	if old[0].state != stepOpen {
		t.Error("take changed the list it was called on")
	}
}

func TestStepListIgnoresAStepItDoesNotHold(t *testing.T) {
	list := testList().take(step.Event{Kind: step.Done, Parent: "other", Step: "Packages"})
	if list.ended() != 0 {
		t.Errorf("ended = %d, want 0", list.ended())
	}
}
