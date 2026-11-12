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
