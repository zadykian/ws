package tui

import (
	"slices"
	"testing"
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
