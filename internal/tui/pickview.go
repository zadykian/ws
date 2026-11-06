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

// groupBox draws a role's box, which follows its steps: "[-]" says that some are on.
func (l pickList) groupBox(parent string) string {
	on, all := 0, 0
	for _, r := range l.rows {
		if r.parent == parent {
			all++
			if r.on() {
				on++
			}
		}
	}
	switch on {
	case 0:
		return "[ ]"
	case all:
		return "[x]"
	}
	return "[-]"
}

// render draws a line. Its box and its mark differ by shape, as NO_COLOR takes the rest.
func (l pickList) render(theme Theme, line pickLine) string {
	r := l.rows[line.row]
	if line.heading {
		return l.groupBox(r.parent) + " " + theme.PaneTitle.Render(r.parent)
	}
	mark, style := r.state.look(theme)
	indent := ""
	if r.parent != "" {
		indent = "  "
	}
	return indent + r.box() + " " + style.Render(mark) + " " + r.name
}

// view draws height lines of the list. The list runs past a window, so that the cursor stays in
// sight, a third of the way down.
func (l pickList) view(theme Theme, height int) string {
	lines := l.lines()
	if len(lines) == 0 {
		return theme.Muted.Render("No step matches.")
	}
	top := max(min(l.at-height/3, len(lines)-height), 0)
	drawn := make([]string, 0, height)
	for i := top; i < min(top+height, len(lines)); i++ {
		gutter := "  "
		if i == l.at {
			gutter = "▸ "
		}
		drawn = append(drawn, gutter+l.render(theme, lines[i]))
	}
	return strings.Join(drawn, "\n")
}
