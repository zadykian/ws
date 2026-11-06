package tui

import (
	"slices"
	"testing"
)

func TestAStageFollowsTheStepsItRuns(t *testing.T) {
	l := choosing(newQueue(testStages()), "one")
	if got := drawn(l, MacSide); got[0] != " 1 [#] · Tools" || got[1] != " 2 [-] · Setup" {
		t.Errorf("list = %q", got)
	}
}

func TestALockedStageRunsWhole(t *testing.T) {
	l := choosing(newQueue(testStages()), "Signing")
	want := []string{"[#]", "[#]", "[#]", "[#]", "[#]", "[#]", "[#]", "[#]", "[x]"}
	if !slices.Equal(boxes(l), want) {
		t.Errorf("boxes = %q, want %q", boxes(l), want)
	}
}
