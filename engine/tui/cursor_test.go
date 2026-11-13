package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestTheCursorStopsAtTheEnds(t *testing.T) {
	cases := []struct {
		msg  tea.KeyPressMsg
		from Cursor
		want Cursor
	}{
		{press('j'), 0, 1},
		{press('j'), 5, 5},
		{press('G'), 0, 5},
		{tea.KeyPressMsg{Code: tea.KeyPgDown}, 1, 5},
		{tea.KeyPressMsg{Code: tea.KeyPgUp}, 5, 1},
	}
	for _, c := range cases {
		if got := c.from.Moved(c.msg, 4, 6); got != c.want {
			t.Errorf("%v from %d = %d, want %d", c.msg, c.from, got, c.want)
		}
	}
}

func TestACursorStaysWithinTheLinesAndTheWindow(t *testing.T) {
	if got := Cursor(7).Within(3); got != 2 {
		t.Errorf("within 3 = %d, want 2", got)
	}
	if got := Cursor(9).Top(6, 20); got != 7 {
		t.Errorf("top = %d, want 7, a third down", got)
	}
	if got := Cursor(19).Top(6, 20); got != 14 {
		t.Errorf("top = %d, want 14, the window's end", got)
	}
}

func TestACursorFollowsItsLineThroughAFilter(t *testing.T) {
	was := []string{"a", "b", "c", "d"}
	if got := Kept(3, was, []string{"a", "d"}); got != 1 {
		t.Errorf("cursor %d, want 1", got)
	}
	if got := Kept(1, was, []string{"a", "c"}); got != 0 {
		t.Errorf("cursor %d, its line went, so the list starts over", got)
	}
}
