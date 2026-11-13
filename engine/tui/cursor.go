package tui

import (
	"slices"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Cursor is a line of a list. A screen holds one per list, so a queue's two sides keep one each.
type Cursor int

// Moved returns the cursor where msg sends it among count lines, page lines to a page.
func (c Cursor) Moved(msg tea.KeyPressMsg, page, count int) Cursor {
	switch {
	case key.Matches(msg, Keys.Up):
		c--
	case key.Matches(msg, Keys.Down):
		c++
	case key.Matches(msg, Keys.PageUp):
		c -= Cursor(page)
	case key.Matches(msg, Keys.PageDown):
		c += Cursor(page)
	case key.Matches(msg, Keys.Top):
		c = 0
	case key.Matches(msg, Keys.Bottom):
		c = Cursor(count - 1)
	}
	return c.Within(count)
}

// Within returns the cursor held to the count lines, or to the first where there are none.
func (c Cursor) Within(count int) Cursor {
	return Cursor(max(min(int(c), count-1), 0))
}

// Top is the first line of a window of height lines over count, so that the cursor stays in sight, a
// third of the way down.
func (c Cursor) Top(height, count int) int {
	return max(min(int(c)-height/3, count-height), 0)
}

// Kept returns the cursor on the line it stood on among was, now that the lines are now. It starts
// the list where that line went.
func Kept[L comparable](at Cursor, was, now []L) Cursor {
	var line L
	if int(at) < len(was) {
		line = was[at]
	}
	return Cursor(max(slices.Index(now, line), 0))
}
