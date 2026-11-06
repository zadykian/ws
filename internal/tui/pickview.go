package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// pickLine is a line of the list: a step, or the heading of the role whose first shown step is row.
type pickLine struct {
	row     int
	heading bool
}

// lines returns the steps, each under its role's heading.
func (l pickList) lines() []pickLine {
	var lines []pickLine
	parent := ""
	for i, r := range l.rows {
		if r.parent != "" && r.parent != parent {
			lines = append(lines, pickLine{row: i, heading: true})
		}
		parent = r.parent
		lines = append(lines, pickLine{row: i})
	}
	return lines
}

// current returns the line under the cursor. A list of no step has none.
func (l pickList) current() (pickLine, bool) {
	lines := l.lines()
	if l.at >= len(lines) {
		return pickLine{}, false
	}
	return lines[l.at], true
}

// moved returns the list with the cursor where msg sends it, page lines to a page.
func (l pickList) moved(msg tea.KeyPressMsg, page int) pickList {
	last := len(l.lines()) - 1
	switch {
	case key.Matches(msg, Keys.Up):
		l.at--
	case key.Matches(msg, Keys.Down):
		l.at++
	case key.Matches(msg, Keys.PageUp):
		l.at -= page
	case key.Matches(msg, Keys.PageDown):
		l.at += page
	case key.Matches(msg, Keys.Top):
		l.at = 0
	case key.Matches(msg, Keys.Bottom):
		l.at = last
	}
	l.at = max(min(l.at, last), 0)
	return l
}
