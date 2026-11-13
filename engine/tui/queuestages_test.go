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

func TestAQueueKeepsItsStagesInTheOrderTheyRun(t *testing.T) {
	var names, waits []string
	var sides []Side
	for _, s := range queueOf(testStages()).stages {
		names, sides, waits = append(names, s.row.name), append(sides, s.side), append(waits, s.row.waitsOn)
	}
	if want := []string{"Tools", "Setup", "Sudo", "Run", "Signing"}; !slices.Equal(names, want) {
		t.Errorf("names = %q, want %q", names, want)
	}
	if want := []Side{MacSide, MacSide, WsSide, WsSide, MacSide}; !slices.Equal(sides, want) {
		t.Errorf("sides = %v, want %v", sides, want)
	}
	if want := []string{"", "Tools", "Setup", "Sudo", "Run"}; !slices.Equal(waits, want) {
		t.Errorf("waits = %q, want %q", waits, want)
	}
}

func TestAStageKeepsItsStepsInAListOfItsOwn(t *testing.T) {
	q := queueOf(testStages())
	for i, want := range map[int]string{0: "", 1: "one two", 3: "base ssh"} {
		var got []string
		for _, r := range q.stages[i].list.rows {
			got = append(got, r.name)
		}
		if joined := strings.Join(got, " "); joined != want {
			t.Errorf("stage %d holds %q, want %q", i, joined, want)
		}
	}
}

func TestARoleOfAStageThatNoScanReadIsOneRow(t *testing.T) {
	rows := queueOf(testStages()).stages[3].list.rows
	if len(rows) != 2 || rows[0].index != unreadIndex || rows[0].form != groupUnread {
		t.Errorf("rows = %+v", rows)
	}
}

// drawnQueue is what a side of q shows, a line each, its colors gone.
func drawnQueue(q queue, side Side) []string {
	var got []string
	for _, line := range q.lines(side) {
		got = append(got, stripped(q.render(testTheme, line)))
	}
	return got
}

func TestAQueueNumbersItsStagesAcrossBothSides(t *testing.T) {
	q := queueOf(testStages())
	mac := []string{" 1 [ ] · Tools", " 2 [ ] · Setup", "     [ ] · one", "     [ ] · two", " 5 [ ] · Signing"}
	if got := drawnQueue(q, MacSide); !slices.Equal(got, mac) {
		t.Errorf("Mac's list = %q, want %q", got, mac)
	}
	ws := []string{" 3 [ ] · Sudo", " 4 [ ] · Run", "     [ ] · base", "     [ ] · ssh"}
	if got := drawnQueue(q, WsSide); !slices.Equal(got, ws) {
		t.Errorf("ws's list = %q, want %q", got, ws)
	}
}

func TestARoleOfAStageOpensUnderItsHeading(t *testing.T) {
	stages := []Stage{{Name: "Run", Nested: []Group{{Name: "base", Steps: []string{"Packages"}}}}}
	got := strings.Join(drawnQueue(queueOf(stages), MacSide), "|")
	if want := " 1 [ ] · Run|     [ ] base|       [ ] · Packages"; got != want {
		t.Errorf("list = %q, want %q", got, want)
	}
}

func TestAFilterLeavesAStepTheHeadingOfItsStage(t *testing.T) {
	got := drawnQueue(queueOf(testStages()).filtered("two"), MacSide)
	if want := []string{"     [ ] Setup", "     [ ] · two"}; !slices.Equal(got, want) {
		t.Errorf("lines = %q, want %q", got, want)
	}
}

func TestAQueueViewDrawsTheWindowUnderTheCursor(t *testing.T) {
	got := stripped(queueOf(testStages()).view(testTheme, MacSide, 2, 2, true))
	if want := "▸      [ ] · one\n       [ ] · two"; got != want {
		t.Errorf("view = %q, want %q", got, want)
	}
}

func TestNarrowingKeepsTheCursorOnItsLine(t *testing.T) {
	q, at := queueOf(testStages()).narrowed(MacSide, 3, "two")
	if q.words != "two" || at != 1 {
		t.Errorf("words %q, cursor %d", q.words, at)
	}
	if _, at = q.narrowed(MacSide, at, ""); at != 3 {
		t.Errorf("cursor = %d, want 3", at)
	}
}
