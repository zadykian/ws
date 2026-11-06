package tui

import (
	"maps"
	"slices"
	"testing"
)

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

func TestARoleUnreadIsChosenByItsName(t *testing.T) {
	got := choosing(newQueue(testStages()), "ssh").partial()["Run"]
	if want := []string{"ssh"}; !slices.Equal(got, want) {
		t.Errorf("partial = %q, want %q", got, want)
	}
}
