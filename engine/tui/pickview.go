package tui

import (
	"strings"

	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// PickLine is a line of the list: a step, or the heading of the role whose first shown step is row.
type PickLine struct {
	Row     int
	Heading bool
}

// shows says that the filter keeps r, by its role or its name.
func (l PickList) shows(r PickRow) bool {
	return strings.Contains(strings.ToLower(r.entry()), strings.ToLower(l.Words))
}

// Named says that the filter's words are in name: a bare group's steps stand under it without a heading.
func (l PickList) Named(name string) bool {
	return strings.Contains(strings.ToLower(name), strings.ToLower(l.Words))
}

// Lines returns what the filter leaves of the side, each step under its role's heading. A stage
// names the steps under it, so that a group called as its stage takes no heading of its own.
func (l PickList) Lines() []PickLine {
	var lines []PickLine
	parent := ""
	for i, r := range l.Rows {
		if !l.shows(r) {
			continue
		}
		if r.Parent != "" && r.Parent != parent && (r.Form != GroupBare || !l.Named(r.Parent)) {
			lines = append(lines, PickLine{Row: i, Heading: true})
		}
		parent = r.Parent
		lines = append(lines, PickLine{Row: i})
	}
	return lines
}

// Current returns the line under the cursor. A filter can leave none.
func (l PickList) Current(at widget.Cursor) (PickLine, bool) {
	lines := l.Lines()
	if int(at) >= len(lines) {
		return PickLine{}, false
	}
	return lines[at], true
}

// Filtered returns the list under words.
func (l PickList) Filtered(words string) PickList {
	l.Words = words
	return l
}

// Narrowed returns the list under words, and the cursor on its line while that shows.
func (l PickList) Narrowed(at widget.Cursor, words string) (PickList, widget.Cursor) {
	was := l.Lines()
	l = l.Filtered(words)
	return l, widget.Kept(at, was, l.Lines())
}

// groupBox draws a role's box, which follows its steps: "[-]" says that some are on.
func (l PickList) groupBox(parent string) string {
	on, all := l.count(func(r PickRow) bool { return r.Parent == parent })
	switch on {
	case 0:
		return "[ ]"
	case all:
		return "[x]"
	}
	return "[-]"
}

// Render draws a line. Its box and its mark differ by shape, as NO_COLOR takes the rest.
func (l PickList) Render(theme widget.Theme, line PickLine) string {
	r := l.Rows[line.Row]
	if line.Heading {
		return l.groupBox(r.Parent) + " " + theme.PaneTitle.Render(r.Parent)
	}
	mark, style := r.State.Look(theme)
	return r.indented() + r.Box() + " " + style.Render(mark) + " " + r.Name
}

// View draws height lines of the list, the cursor's gutter always.
func (l PickList) View(theme widget.Theme, height int, at widget.Cursor) string {
	lines := l.Lines()
	return Windowed(theme, height, at, true, len(lines), func(i int) string { return l.Render(theme, lines[i]) })
}

// Windowed draws height of count lines, each by draw, the cursor's gutter where shown. The lines run
// past a window, so that the cursor stays in sight.
func Windowed(theme widget.Theme, height int, at widget.Cursor, shown bool, count int, draw func(int) string) string {
	if count == 0 {
		return theme.Muted.Render("No step matches.")
	}
	top := at.Top(height, count)
	drawn := make([]string, 0, height)
	for i := top; i < min(top+height, count); i++ {
		gutter := "  "
		if shown && i == int(at) {
			gutter = "▸ "
		}
		drawn = append(drawn, gutter+draw(i))
	}
	return strings.Join(drawn, "\n")
}
