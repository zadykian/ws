package tui

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/tui/internal/list"
)

func TestAQueueChoiceHoldsTheStagesOnlyWhereItRuns(t *testing.T) {
	s := queueSent(testQueueScreen(), list.ScannedMsg{}, press('n'), press(' '))
	for next, want := range map[Next][]string{NextRun: {"Tools"}, NextCheck: {"Tools"}, NextConfigure: nil, NextState: nil, NextLeave: nil} {
		if got := s.queueChoice(next); got.Next != next || !slices.Equal(got.Only, want) {
			t.Errorf("%d: choice = %+v, want only %q", next, got, want)
		}
	}
	if got := s.queueChoice(NextState); got.Side != s.list.side {
		t.Errorf("side = %q, want %q", got.Side, s.list.side)
	}
}
