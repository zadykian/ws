package tui

import (
	"slices"
	"strings"
	"testing"
)

// testStages is a queue across both sides: a stage of steps, a stage of roles no scan has read.
func testStages() []Stage {
	return []Stage{
		{Name: "Tools"},
		{Name: "Setup", Waits: "Tools", Nested: []Group{{Name: "Setup", Steps: []string{"one", "two"}}}},
		{Name: "Sudo", Side: WsSide, Waits: "Setup"},
		{Name: "Run", Side: WsSide, Waits: "Sudo", Nested: []Group{{Name: "base"}, {Name: "ssh"}}},
		{Name: "Signing", Waits: "Run"},
	}
}

// drawn is what a side of l shows, a line each, its colors gone.
func drawn(l pickList, side Side) []string {
	l.side = side
	var got []string
	for _, line := range l.lines() {
		got = append(got, stripped(l.render(testTheme, line)))
	}
	return got
}

func TestAStageIsNumberedAndItsStepsFollowIt(t *testing.T) {
	var numbers []int
	var names []string
	for _, r := range newQueue(testStages()).rows {
		numbers, names = append(numbers, r.number), append(names, r.name)
	}
	if want := []int{1, 2, 0, 0, 3, 4, 0, 0, 5}; !slices.Equal(numbers, want) {
		t.Errorf("numbers = %v, want %v", numbers, want)
	}
	if want := []string{"Tools", "Setup", "one", "two", "Sudo", "Run", "base", "ssh", "Signing"}; !slices.Equal(names, want) {
		t.Errorf("names = %q, want %q", names, want)
	}
}

func TestAQueueNumbersItsStagesAcrossBothSides(t *testing.T) {
	l := newQueue(testStages())
	mac := []string{" 1 [ ] · Tools", " 2 [ ] · Setup", "     [ ] · one", "     [ ] · two", " 5 [ ] · Signing"}
	if got := drawn(l, MacSide); !slices.Equal(got, mac) {
		t.Errorf("Mac's list = %q, want %q", got, mac)
	}
	ws := []string{" 3 [ ] · Sudo", " 4 [ ] · Run", "     [ ] · base", "     [ ] · ssh"}
	if got := drawn(l, WsSide); !slices.Equal(got, ws) {
		t.Errorf("ws's list = %q, want %q", got, ws)
	}
}

func TestARoleOfAStageOpensUnderItsHeading(t *testing.T) {
	stages := []Stage{{Name: "Run", Nested: []Group{{Name: "base", Steps: []string{"Packages"}}}}}
	got := strings.Join(drawn(newQueue(stages), MacSide), "|")
	if want := " 1 [ ] · Run|     [ ] base|       [ ] · Packages"; got != want {
		t.Errorf("list = %q, want %q", got, want)
	}
}

func TestAFilterLeavesAStepTheHeadingOfItsStage(t *testing.T) {
	got := newQueue(testStages()).filtered("two").lines()
	if want := []pickLine{{3, true}, {3, false}}; !slices.Equal(got, want) {
		t.Errorf("lines = %v, want %v", got, want)
	}
}

func TestAWaitOfTheSameNameStaysWithItsStage(t *testing.T) {
	first := pickRow{stepRow: stepRow{name: "a"}, stage: "one"}
	second := pickRow{stepRow: stepRow{name: "a"}, stage: "two"}
	if first.sameChoice(second) {
		t.Errorf("%+v and %+v are one choice", first, second)
	}
}
