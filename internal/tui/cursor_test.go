package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestACursorMovesAndStopsAtTheEnds(t *testing.T) {
	pgdown := tea.KeyPressMsg{Code: tea.KeyPgDown}
	for _, c := range []struct {
		name string
		from cursor
		msg  tea.KeyPressMsg
		want cursor
	}{
		{"down", 1, press('j'), 2},
		{"up at the top", 0, press('k'), 0},
		{"down at the end", 4, press('j'), 4},
		{"bottom", 0, press('G'), 4},
		{"page down", 1, pgdown, 4},
	} {
		if got := c.from.moved(c.msg, 3, 5); got != c.want {
			t.Errorf("%s: cursor %d, want %d", c.name, got, c.want)
		}
	}
}

func TestACursorStaysWithinTheLinesAndTheWindow(t *testing.T) {
	if got := cursor(7).within(3); got != 2 {
		t.Errorf("within 3 = %d, want 2", got)
	}
	if got := cursor(9).top(6, 20); got != 7 {
		t.Errorf("top = %d, want 7, a third down", got)
	}
	if got := cursor(19).top(6, 20); got != 14 {
		t.Errorf("top = %d, want 14, the window's end", got)
	}
}

func TestACursorFollowsItsLineThroughAFilter(t *testing.T) {
	was := []string{"a", "b", "c", "d"}
	if got := kept(3, was, []string{"a", "d"}); got != 1 {
		t.Errorf("cursor %d, want 1", got)
	}
	if got := kept(1, was, []string{"a", "c"}); got != 0 {
		t.Errorf("cursor %d, its line went, so the list starts over", got)
	}
}
