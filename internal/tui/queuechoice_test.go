package tui

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/step"
)

// atRow returns l turned to the side of the row of that name, and the cursor on the row.
func atRow(l pickList, name string) (pickList, cursor) {
	row := slices.IndexFunc(l.rows, func(r pickRow) bool { return r.name == name })
	l.side = l.rows[row].side
	return l, cursor(slices.Index(l.lines(), pickLine{row: row}))
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

func TestAStageRunsWhatTheScanFound(t *testing.T) {
	stages := []Stage{{Name: "Run", Nested: []Group{{Name: "base", Steps: []string{"one", "two"}}}}}
	l := newQueue(stages).take(planned("base", "one", 0, oneChange("a"))).take(planned("base", "two", 1)).
		take(planned("", "Run", 0)).opened()
	if want := []string{"[x]", "[x]", "[ ]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
}

func TestALeftOutWarningStaysAmongOneStagesRoles(t *testing.T) {
	stages := []Stage{
		{Name: "Setup", Nested: []Group{{Name: "Setup", Steps: []string{"one"}}}},
		{Name: "Run", Nested: []Group{{Name: "base", Steps: []string{"two"}}, {Name: "ssh", Steps: []string{"three"}}}},
	}
	l := newQueue(stages)
	for _, e := range []step.Event{planned("Setup", "one", 0, oneChange("a")), planned("base", "two", 0, oneChange("a")), planned("ssh", "three", 0, oneChange("a"))} {
		l = l.take(e)
	}
	if got, want := choosing(l, "three").passedOver(), []string{"base"}; !slices.Equal(got, want) {
		t.Errorf("passed over = %q, want %q", got, want)
	}
}
