package tui

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// pickLine is a line of the list: a step, or the heading of the role whose first shown step is row.
type pickLine struct {
	row     int
	heading bool
}

// shows says that the filter keeps r, by its role or its name.
func (l pickList) shows(r pickRow) bool {
	return strings.Contains(strings.ToLower(r.entry()), strings.ToLower(l.words))
}

// lines returns what the filter leaves, each step under its role's heading.
func (l pickList) lines() []pickLine {
	var lines []pickLine
	parent := ""
	for i, r := range l.rows {
		if !l.shows(r) {
			continue
		}
		if r.parent != "" && r.parent != parent {
			lines = append(lines, pickLine{row: i, heading: true})
		}
		parent = r.parent
		lines = append(lines, pickLine{row: i})
	}
	return lines
}

// current returns the line under the cursor. A filter can leave none.
func (l pickList) current() (pickLine, bool) {
	lines := l.lines()
	if l.at >= len(lines) {
		return pickLine{}, false
	}
	return lines[l.at], true
}

// filtered returns the list under words, the cursor on its line while that shows.
func (l pickList) filtered(words string) pickList {
	was, _ := l.current()
	l.words = words
	l.at = max(slices.Index(l.lines(), was), 0)
	return l
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
