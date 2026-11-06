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
