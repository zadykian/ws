package queue

import (
	"maps"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui/internal/list"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// queueBoxes draws each row's box in order, the stages' own among them, as the eye reads a queue.
func queueBoxes(q queue) []string {
	var drawn []string
	for _, s := range q.stages {
		drawn = append(drawn, s.row.Box())
		for _, r := range s.list.Rows {
			drawn = append(drawn, r.Box())
		}
	}
	return drawn
}

func TestAStagesBoxTurnsItsSteps(t *testing.T) {
	side, at := "Mac", widget.Cursor(1)
	q := queueOf(testStages()).toggled(side, at)
	if got := queueBoxes(q)[1:4]; !slices.Equal(got, []string{"[x]", "[x]", "[x]"}) {
		t.Errorf("boxes = %q", got)
	}
	if got := queueBoxes(q.toggled(side, at))[1:4]; !slices.Equal(got, []string{"[ ]", "[ ]", "[ ]"}) {
		t.Errorf("boxes = %q", got)
	}
}

func TestAStepsBoxTurnsAloneAndALockedOneHolds(t *testing.T) {
	q := queueOf(testStages()).toggled("Mac", 3)
	if got, want := queueBoxes(q)[:4], []string{"[#]", "[x]", "[ ]", "[x]"}; !slices.Equal(got, want) {
		t.Errorf("boxes = %q, want %q", got, want)
	}
	if got := q.toggled("Mac", 0); !slices.Equal(queueBoxes(got), queueBoxes(q)) {
		t.Errorf("a locked stage turned: %q", queueBoxes(got))
	}
	if got := queueBoxes(queueOf(testStages()).toggled("ws", 0))[:5]; !slices.Equal(got, []string{"[#]", "[#]", "[#]", "[#]", "[x]"}) {
		t.Errorf("a stage of the ws side left its lock: %q", got)
	}
}

// queueChoosing returns q with the rows of those names chosen, and no other.
func queueChoosing(q queue, names ...string) queue {
	q.stages = slices.Clone(q.stages)
	for i, s := range q.stages {
		q.stages[i].row.Chosen = slices.Contains(names, s.row.Name)
		q.stages[i].list.Rows = slices.Clone(s.list.Rows)
		for j, r := range s.list.Rows {
			q.stages[i].list.Rows[j].Chosen = slices.Contains(names, r.Name)
		}
	}
	return q.settled()
}

func TestAStageFollowsTheStepsItRuns(t *testing.T) {
	q := queueChoosing(queueOf(testStages()), "one")
	if got := drawnQueue(q, "Mac"); got[0] != " 1 [#] · Tools" || got[1] != " 2 [-] · Setup" {
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
	if !slices.Equal(queueBoxes(q), want) || len(q.partial()) != 0 {
		t.Errorf("boxes = %q, partial = %q, want %q", queueBoxes(q), q.partial(), want)
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
	stages := []Stage{{Name: "Run", Side: "Mac", Nested: []list.Group{{Name: "base"}}}}
	if got := queueOf(stages).take(planned("", "Run", 0)).opened().only(); len(got) != 0 {
		t.Errorf("only = %q", got)
	}
}

func TestAStageDueRunsWholeWhereItsStepsReadClean(t *testing.T) {
	stages := []Stage{{Name: "Run", Side: "Mac", Nested: []list.Group{{Name: "base", Steps: []string{"one", "two"}}}}}
	q := queueOf(stages).take(planned("base", "one", 0)).take(planned("base", "two", 1)).
		take(planned("", "Run", 0, oneChange("a"))).opened()
	if want := []string{"[x]", "[x]", "[x]"}; !slices.Equal(queueBoxes(q), want) {
		t.Errorf("boxes = %q, want %q", queueBoxes(q), want)
	}
}

func TestAStageRunsWhatTheScanFound(t *testing.T) {
	stages := []Stage{{Name: "Run", Side: "Mac", Nested: []list.Group{{Name: "base", Steps: []string{"one", "two"}}}}}
	q := queueOf(stages).take(planned("base", "one", 0, oneChange("a"))).take(planned("base", "two", 1)).
		take(planned("", "Run", 0)).opened()
	if want := []string{"[x]", "[x]", "[ ]"}; !slices.Equal(queueBoxes(q), want) {
		t.Errorf("boxes = %q, want %q", queueBoxes(q), want)
	}
}

func TestALeftOutWarningStaysAmongOneStagesRoles(t *testing.T) {
	stages := []Stage{
		{Name: "Setup", Side: "Mac", Nested: []list.Group{{Name: "Setup", Steps: []string{"one"}}}},
		{Name: "Run", Side: "Mac", Nested: []list.Group{{Name: "base", Steps: []string{"two"}}, {Name: "ssh", Steps: []string{"three"}}}},
	}
	q := queueOf(stages)
	q = q.take(planned("Setup", "one", 0, oneChange("a"))).take(planned("base", "two", 0, oneChange("a")))
	q = q.take(planned("ssh", "three", 0, oneChange("a")))
	if got, want := queueChoosing(q, "three").passedOver(1), []string{"base"}; !slices.Equal(got, want) {
		t.Errorf("passed over = %q, want %q", got, want)
	}
}

func TestABulkChoiceLeavesWhatTheFilterHides(t *testing.T) {
	q, _ := queueOf(testStages()).narrowed("Mac", 0, "two")
	q = q.chose(func(list.PickRow) bool { return true })
	if want := []string{"[#]", "[x]", "[ ]", "[x]"}; !slices.Equal(queueBoxes(q)[:4], want) {
		t.Errorf("boxes = %q, want %q", queueBoxes(q)[:4], want)
	}
}

func TestAQueuePassesOverTheRolesOfEachStage(t *testing.T) {
	stages := []Stage{
		{Name: "Run", Side: "Mac", Nested: []list.Group{{Name: "base", Steps: []string{"one"}}, {Name: "ssh", Steps: []string{"two"}}}},
		{Name: "Sudo", Side: "ws", Nested: []list.Group{{Name: "net", Steps: []string{"three"}}, {Name: "dns", Steps: []string{"four"}}}},
	}
	q := queueOf(stages)
	for _, e := range []step.Event{planned("base", "one", 0, oneChange("a")), planned("net", "three", 0, oneChange("a"))} {
		q = q.take(e)
	}
	if got, want := queueChoosing(q, "two", "four").passed(), []string{"base", "net"}; !slices.Equal(got, want) {
		t.Errorf("passed = %q, want %q", got, want)
	}
}
