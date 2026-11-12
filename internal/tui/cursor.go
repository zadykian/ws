package tui

import (
	"slices"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// cursor is a line of a list. A screen holds one per list, so a queue's two sides keep one each.
type cursor int

// moved returns the cursor where msg sends it among count lines, page lines to a page.
func (c cursor) moved(msg tea.KeyPressMsg, page, count int) cursor {
	switch {
	case key.Matches(msg, Keys.Up):
		c--
	case key.Matches(msg, Keys.Down):
		c++
	case key.Matches(msg, Keys.PageUp):
		c -= cursor(page)
	case key.Matches(msg, Keys.PageDown):
		c += cursor(page)
	case key.Matches(msg, Keys.Top):
		c = 0
	case key.Matches(msg, Keys.Bottom):
		c = cursor(count - 1)
	}
	return c.within(count)
}

// within returns the cursor held to the count lines, or to the first where there are none.
func (c cursor) within(count int) cursor {
	return cursor(max(min(int(c), count-1), 0))
}

// top is the first line of a window of height lines over count, so that the cursor stays in sight, a
// third of the way down.
func (c cursor) top(height, count int) int {
	return max(min(int(c)-height/3, count-height), 0)
}

// kept returns the cursor on the line it stood on among was, now that the lines are now. It starts
// the list where that line went.
func kept[L comparable](at cursor, was, now []L) cursor {
	var line L
	if int(at) < len(was) {
		line = was[at]
	}
	return cursor(max(slices.Index(now, line), 0))
}
