package tui

import (
	"maps"
	"slices"
	"testing"
)

// queueBoxes draws each row's box in order, the stages' own among them, as the eye reads a queue.
func queueBoxes(q queue) []string {
	var drawn []string
	for _, s := range q.stages {
		drawn = append(drawn, s.row.box())
		for _, r := range s.list.rows {
			drawn = append(drawn, r.box())
		}
	}
	return drawn
}

// atStage returns the side and the cursor on the line of the stage called name.
func atStage(q queue, name string) (Side, cursor) {
	i := slices.IndexFunc(q.stages, func(s queueStage) bool { return s.row.name == name })
	side := q.stages[i].side
	return side, cursor(slices.Index(q.lines(side), queueLine{stage: i, head: true}))
}

func TestAStagesBoxTurnsItsSteps(t *testing.T) {
	q := queueOf(testStages())
	side, at := atStage(q, "Setup")
	q = q.toggled(side, at)
	if got := queueBoxes(q)[1:4]; !slices.Equal(got, []string{"[x]", "[x]", "[x]"}) {
		t.Errorf("boxes = %q", got)
	}
	if got := queueBoxes(q.toggled(side, at))[1:4]; !slices.Equal(got, []string{"[ ]", "[ ]", "[ ]"}) {
		t.Errorf("boxes = %q", got)
	}
}

func TestAStepsBoxTurnsAloneAndALockedOneHolds(t *testing.T) {
	q := queueOf(testStages()).toggled(MacSide, 3)
	if got, want := queueBoxes(q)[:4], []string{"[#]", "[x]", "[ ]", "[x]"}; !slices.Equal(got, want) {
		t.Errorf("boxes = %q, want %q", got, want)
	}
	if got := q.toggled(MacSide, 0); !slices.Equal(queueBoxes(got), queueBoxes(q)) {
		t.Errorf("a locked stage turned: %q", queueBoxes(got))
	}
}

// queueChoosing returns q with the rows of those names chosen, and no other.
func queueChoosing(q queue, names ...string) queue {
	q.stages = slices.Clone(q.stages)
	for i, s := range q.stages {
		q.stages[i].row.chosen = slices.Contains(names, s.row.name)
		q.stages[i].list.rows = slices.Clone(s.list.rows)
		for j, r := range s.list.rows {
			q.stages[i].list.rows[j].chosen = slices.Contains(names, r.name)
		}
	}
	return q.settled()
}

func TestAStageFollowsTheStepsItRuns(t *testing.T) {
	q := queueChoosing(queueOf(testStages()), "one")
	if got := drawnQueue(q, MacSide); got[0] != " 1 [#] · Tools" || got[1] != " 2 [-] · Setup" {
		t.Errorf("list = %q", got)
	}
	if got, want := q.only(), []string{"Tools", "Setup"}; !slices.Equal(got, want) {
		t.Errorf("only = %q, want %q", got, want)
	}
	parts := map[string][]string{"Setup": {"Setup/one"}}
	if !maps.EqualFunc(q.partial(), parts, slices.Equal) {
		t.Errorf("partial = %q", q.partial())
	}
}

func TestALockedStageRunsWhole(t *testing.T) {
	q := queueChoosing(queueOf(testStages()), "Signing")
	want := []string{"[#]", "[#]", "[#]", "[#]", "[#]", "[#]", "[#]", "[#]", "[x]"}
	if !slices.Equal(queueBoxes(q), want) {
		t.Errorf("boxes = %q, want %q", queueBoxes(q), want)
	}
	if len(q.partial()) != 0 {
		t.Errorf("partial = %q", q.partial())
	}
}

func TestAStageRunsWholeByATickAndNotByALockAlone(t *testing.T) {
	for _, tc := range []struct {
		names, want []string
	}{
		{[]string{"Signing"}, nil},
		{[]string{"ssh"}, nil},
		{[]string{"Signing", "ssh"}, []string{"Run"}},
		{[]string{"base", "ssh"}, []string{"Run"}},
		{[]string{"one", "base", "ssh"}, []string{"Setup", "Run"}},
	} {
		if got := queueChoosing(queueOf(testStages()), tc.names...).ticked(); !slices.Equal(got, tc.want) {
			t.Errorf("%q ticks %q, want %q", tc.names, got, tc.want)
		}
	}
}

func TestARoleUnreadIsChosenByItsName(t *testing.T) {
	got := queueChoosing(queueOf(testStages()), "ssh").partial()["Run"]
	if want := []string{"ssh"}; !slices.Equal(got, want) {
		t.Errorf("partial = %q, want %q", got, want)
	}
}

func TestAQueueCountsItsStagesOnAndReached(t *testing.T) {
	q := queueChoosing(queueOf(testStages()), "Signing").take(planned("", "Tools", 0))
	if on, scanned, all := q.count(); on != 5 || scanned != 1 || all != 5 {
		t.Errorf("count = %d, %d, %d", on, scanned, all)
	}
}

func TestWhatNoScanReachedStartsChosen(t *testing.T) {
	q := queueOf(testStages()).take(planned("", "Tools", 0)).opened()
	if want := slices.Concat(slices.Repeat([]string{"[#]"}, 8), []string{"[x]"}); !slices.Equal(queueBoxes(q), want) {
		t.Errorf("boxes = %q, want %q", queueBoxes(q), want)
	}
}

func TestRolesUnreadFollowAStageThatFoundNothing(t *testing.T) {
	stages := []Stage{{Name: "Run", Nested: []Group{{Name: "base"}}}}
	if got := queueOf(stages).take(planned("", "Run", 0)).opened().only(); len(got) != 0 {
		t.Errorf("only = %q", got)
	}
}

func TestAStageDueRunsWholeWhereItsStepsReadClean(t *testing.T) {
	stages := []Stage{{Name: "Run", Nested: []Group{{Name: "base", Steps: []string{"one", "two"}}}}}
	q := queueOf(stages).take(planned("base", "one", 0)).take(planned("base", "two", 1)).
		take(planned("", "Run", 0, oneChange("a"))).opened()
	if want := []string{"[x]", "[x]", "[x]"}; !slices.Equal(queueBoxes(q), want) {
		t.Errorf("boxes = %q, want %q", queueBoxes(q), want)
	}
}
