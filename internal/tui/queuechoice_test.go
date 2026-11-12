package tui

import (
	"maps"
	"slices"
	"testing"

	"github.com/zadykian/ws/step"
)

// atRow returns l with the cursor on the row of that name, on its side.
func atRow(l pickList, name string) pickList {
	row := slices.IndexFunc(l.rows, func(r pickRow) bool { return r.name == name })
	l.side = l.rows[row].side
	l.at = cursor(slices.Index(l.lines(), pickLine{row: row}))
	return l
}

func TestAStageFollowsTheStepsItRuns(t *testing.T) {
	l := choosing(newQueue(testStages()), "one")
	if got := drawn(l, MacSide); got[0] != " 1 [#] · Tools" || got[1] != " 2 [-] · Setup" {
		t.Errorf("list = %q", got)
	}
	if got, want := l.only(), []string{"Tools", "Setup"}; !slices.Equal(got, want) {
		t.Errorf("only = %q, want %q", got, want)
	}
	parts := map[string][]string{"Setup": {"Setup/one"}}
	if !maps.EqualFunc(l.partial(), parts, slices.Equal) {
		t.Errorf("partial = %q", l.partial())
	}
}

func TestALockedStageRunsWhole(t *testing.T) {
	l := choosing(newQueue(testStages()), "Signing")
	want := []string{"[#]", "[#]", "[#]", "[#]", "[#]", "[#]", "[#]", "[#]", "[x]"}
	if !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
	if len(l.partial()) != 0 {
		t.Errorf("partial = %q", l.partial())
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
		if got := choosing(newQueue(testStages()), tc.names...).ticked(); !slices.Equal(got, tc.want) {
			t.Errorf("%q ticks %q, want %q", tc.names, got, tc.want)
		}
	}
}

func TestARoleUnreadIsChosenByItsName(t *testing.T) {
	got := choosing(newQueue(testStages()), "ssh").partial()["Run"]
	if want := []string{"ssh"}; !slices.Equal(got, want) {
		t.Errorf("partial = %q, want %q", got, want)
	}
}

func TestAStagesBoxTurnsItsSteps(t *testing.T) {
	l := atRow(newQueue(testStages()), "Setup").toggled()
	if got := boxes(l)[1:4]; !slices.Equal(got, []string{"[x]", "[x]", "[x]"}) {
		t.Errorf("boxes = %q", got)
	}
	if got := boxes(l.toggled())[1:4]; !slices.Equal(got, []string{"[ ]", "[ ]", "[ ]"}) {
		t.Errorf("boxes = %q", got)
	}
}

func TestALockedStageHoldsItsBox(t *testing.T) {
	l := atRow(choosing(newQueue(testStages()), "Signing"), "Run").toggled()
	if !l.rows[5].on() || !l.rows[6].on() {
		t.Errorf("rows = %+v", l.rows)
	}
}

func TestWhatNoScanReachedStartsChosen(t *testing.T) {
	l := newQueue(testStages()).take(planned("", "Tools", 0)).opened()
	if want := slices.Concat(slices.Repeat([]string{"[#]"}, 8), []string{"[x]"}); !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
}

func TestRolesUnreadFollowAStageThatFoundNothing(t *testing.T) {
	stages := []Stage{{Name: "Run", Nested: []Group{{Name: "base"}}}}
	if got := newQueue(stages).take(planned("", "Run", 0)).opened().only(); len(got) != 0 {
		t.Errorf("only = %q", got)
	}
}

func TestAStageDueRunsWholeWhereItsStepsReadClean(t *testing.T) {
	stages := []Stage{{Name: "Run", Nested: []Group{{Name: "base", Steps: []string{"one", "two"}}}}}
	l := newQueue(stages).take(planned("base", "one", 0)).take(planned("base", "two", 1)).
		take(planned("", "Run", 0, oneChange("a"))).opened()
	if want := []string{"[x]", "[x]", "[x]"}; !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
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
