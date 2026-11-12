package tui

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/step"
)

func TestAnEventNamesAStepNoScanHad(t *testing.T) {
	l := newQueue(testStages()).take(planned("", "Run", 3)).take(planned("base", "Packages", 0, oneChange("a")))
	l = l.take(planned("base", "Locale", 1))
	got := drawn(l, WsSide)
	want := []string{" 3 [#] · Sudo", " 4 [-] ✓ Run", "     [-] base", "       [x] ● Packages", "       [ ] ✓ Locale", "     [ ] · ssh"}
	if !slices.Equal(got, want) {
		t.Errorf("list = %q, want %q", got, want)
	}
}

func TestAnEventOfAStageNamedAsItsGroupJoinsItsSteps(t *testing.T) {
	l := newQueue(testStages()).take(planned("Setup", "three", 2))
	if got := drawn(l, MacSide)[4]; got != "     [ ] ✓ three" {
		t.Errorf("row = %q", got)
	}
}

func TestAnEventOutsideEveryStageMovesNothing(t *testing.T) {
	was := newQueue(testStages())
	if got := was.take(planned("other", "x", 0)); len(got.rows) != len(was.rows) {
		t.Errorf("rows = %+v", got.rows)
	}
}

func TestAReportNamesAStagesStepsInPlaceOfItsRoles(t *testing.T) {
	sections := []StateSection{{Name: "base", Reports: []step.Report{
		{Step: "Packages"},
		{Step: "Locale", Changes: []step.Pending{{Text: "sets the locale"}}},
		{Step: "Time", After: "Locale"},
		{Step: "Fans", Items: []step.Item{{Kind: step.CheckItem, Conflict: "no sensor"}}},
	}}}
	l := newQueue(testStages()).reported("Run", sections)
	want := []string{" 3 [#] · Sudo", " 4 [-] · Run", "     [-] base", "       [ ] ✓ Packages", "       [x] ● Locale",
		"       [x] – Time", "       [ ] ✗ Fans"}
	if got := drawn(l, WsSide); !slices.Equal(got, want) {
		t.Errorf("list = %q, want %q", got, want)
	}
	if got := l.rows[7].said; len(got) != 1 || got[0].Text != "sets the locale" {
		t.Errorf("said = %+v", got)
	}
	if got := l.rows[9].said; len(got) != 1 || got[0].Kind != Warned || got[0].Text != "no sensor" {
		t.Errorf("said = %+v", got)
	}
}

func TestARunMovesTheMarksAndLeavesTheBoxes(t *testing.T) {
	l := choosing(newQueue(testStages()), "Tools")
	l, line := l.follow(step.Event{Kind: step.Started, Step: "Sudo", Index: 2})
	if l.side != WsSide || line != 0 {
		t.Errorf("side %d, line %d", l.side, line)
	}
	l, line = l.follow(step.Event{Kind: step.Done, Step: "Tools", Index: 0, LeftOut: true})
	if line >= 0 {
		t.Errorf("line %d: a step done takes no cursor", line)
	}
	if got := drawn(l, MacSide)[0]; got != " 1 [x] ○ Tools" {
		t.Errorf("row = %q", got)
	}
}

func TestARunLearnsTheStepsOfARoleAndOpensOneAgain(t *testing.T) {
	event := step.Event{Kind: step.Planned, Parent: "ssh", Step: "Keys"}
	l, _ := newQueue(testStages()).follow(event)
	if got := drawn(l, WsSide)[3:]; !slices.Equal(got, []string{"     [ ] ssh", "       [ ] · Keys"}) {
		t.Errorf("list = %q", got)
	}
	event.Kind = step.Done
	if got := followed(l, event).rows[l.place(event)].state; got != stepSame {
		t.Errorf("state = %d", got)
	}
	event.Kind = step.Planned
	if got := followed(l, event).rows[l.place(event)].state; got != stepOpen {
		t.Errorf("state = %d", got)
	}
}

// followed is l after e, the cursor aside.
func followed(l pickList, e step.Event) pickList {
	l, _ = l.follow(e)
	return l
}
