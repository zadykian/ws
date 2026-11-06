package tui

import (
	"slices"
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
