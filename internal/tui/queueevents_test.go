package tui

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/step"
)

func TestAnEventOutsideEveryStageMovesNothing(t *testing.T) {
	was := queueOf(testStages()).hear(Line{Said, "x"})
	got := was.take(planned("other", "x", 0))
	if !slices.Equal(drawnQueue(got, MacSide), drawnQueue(was, MacSide)) || len(got.heard) != 0 {
		t.Errorf("queue = %+v", got)
	}
	if got = was.take(planned("", "x", 9)); !slices.Equal(drawnQueue(got, WsSide), drawnQueue(was, WsSide)) {
		t.Errorf("queue = %+v", got)
	}
}

func TestAStageTakesTheWordsTheScanSaidBeforeItsStepEnded(t *testing.T) {
	q := queueOf(testStages()).hear(Line{Said, "Tools: found it"})
	q = q.take(planned("", "Tools", 0, oneChange("a")))
	if said := q.stages[0].row.said; len(said) != 1 || said[0].Text != "found it" || len(q.heard) != 0 {
		t.Errorf("said = %+v, heard = %+v", said, q.heard)
	}
}

func TestAnEventNamesAStepNoScanHad(t *testing.T) {
	q := queueOf(testStages()).take(planned("", "Run", 3)).take(planned("base", "Packages", 0, oneChange("a")))
	q = q.take(planned("base", "Locale", 1))
	got := drawnQueue(q, WsSide)
	want := []string{" 3 [#] · Sudo", " 4 [-] ✓ Run", "     [-] base", "       [x] ● Packages", "       [ ] ✓ Locale",
		"     [ ] · ssh"}
	if !slices.Equal(got, want) {
		t.Errorf("list = %q, want %q", got, want)
	}
}

func TestAnEventOfAStageNamedAsItsGroupJoinsItsSteps(t *testing.T) {
	q := queueOf(testStages()).take(planned("Setup", "three", 2))
	if got := drawnQueue(q, MacSide)[4]; got != "     [ ] ✓ three" {
		t.Errorf("row = %q", got)
	}
}

func TestAnEventOfAStageWithNoGroupOfItsNameJoinsItsList(t *testing.T) {
	q := queueOf(testStages()).take(planned("Run", "Sudo again", 2))
	if got := drawnQueue(q, WsSide)[2]; got != "     [ ] ✓ Sudo again" {
		t.Errorf("row = %q", got)
	}
}

func TestAReportNamesAStagesStepsInPlaceOfItsRoles(t *testing.T) {
	sections := []StateSection{{Name: "base", Reports: []step.Report{
		{Step: "Packages"},
		{Step: "Locale", Changes: []step.Pending{{Text: "sets the locale"}}},
		{Step: "Time", After: "Locale"},
		{Step: "Fans", Items: []step.Item{{Kind: step.CheckItem, Conflict: "no sensor"}}},
	}}}
	q := queueOf(testStages()).reported("Run", sections)
	want := []string{" 3 [#] · Sudo", " 4 [-] · Run", "     [-] base", "       [ ] ✓ Packages", "       [x] ● Locale",
		"       [x] – Time", "       [ ] ✗ Fans"}
	if got := drawnQueue(q, WsSide); !slices.Equal(got, want) {
		t.Errorf("list = %q, want %q", got, want)
	}
	rows := q.stages[3].list.rows
	if got := rows[1].said; len(got) != 1 || got[0].Text != "sets the locale" {
		t.Errorf("said = %+v", got)
	}
	if got := rows[3].said; len(got) != 1 || got[0].Kind != Warned || got[0].Text != "no sensor" {
		t.Errorf("said = %+v", got)
	}
}

func TestAReportKeepsTheWordsOfTheFilter(t *testing.T) {
	sections := []StateSection{{Name: "Run", Reports: []step.Report{{Step: "Packages"}, {Step: "Locale"}}}}
	q := queueOf(testStages()).filtered("loc").reported("Run", sections)
	if got, want := drawnQueue(q, WsSide), []string{"     [ ] Run", "     [ ] ✓ Locale"}; !slices.Equal(got, want) {
		t.Errorf("list = %q, want %q", got, want)
	}
}

func TestARunMovesTheMarksAndLeavesTheBoxes(t *testing.T) {
	q := queueOf(testStages()).take(planned("", "Tools", 0, oneChange("a")))
	q, side, line := q.follow(step.Event{Kind: step.Started, Step: "Sudo", Index: 2})
	if side != WsSide || line != 0 {
		t.Errorf("side %d, line %d", side, line)
	}
	q, _, line = q.follow(step.Event{Kind: step.Done, Step: "Tools", Index: 0, LeftOut: true})
	if line >= 0 {
		t.Errorf("line %d: a step done takes no cursor", line)
	}
	if got := drawnQueue(q, MacSide)[0]; got != " 1 [x] ○ Tools" {
		t.Errorf("row = %q", got)
	}
}

func TestARunLearnsTheStepsOfARoleAndOpensOneAgain(t *testing.T) {
	event := step.Event{Kind: step.Planned, Parent: "ssh", Step: "Keys"}
	q, side, line := queueOf(testStages()).follow(event)
	if got := drawnQueue(q, WsSide)[3:]; !slices.Equal(got, []string{"     [ ] ssh", "       [ ] · Keys"}) {
		t.Errorf("list = %q", got)
	}
	if side != WsSide || line != 4 {
		t.Errorf("side %d, line %d: the cursor is on the step", side, line)
	}
	event.Kind = step.Done
	q, _, _ = q.follow(event)
	if got := drawnQueue(q, WsSide)[4]; got != "       [ ] ✓ Keys" {
		t.Errorf("row = %q", got)
	}
	event.Kind = step.Planned
	q, _, _ = q.follow(event)
	if got := drawnQueue(q, WsSide)[4]; got != "       [ ] · Keys" {
		t.Errorf("row = %q", got)
	}
}
