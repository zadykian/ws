package tui

import (
	"slices"
	"testing"
)

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

func TestAWaitOfTheSameNameStaysWithItsStage(t *testing.T) {
	first := pickRow{stepRow: stepRow{name: "a"}, stage: "one"}
	second := pickRow{stepRow: stepRow{name: "a"}, stage: "two"}
	if first.sameChoice(second) {
		t.Errorf("%+v and %+v are one choice", first, second)
	}
}
