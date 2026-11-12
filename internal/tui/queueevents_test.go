package tui

import (
	"slices"
	"testing"
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
