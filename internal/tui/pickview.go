package tui

import "strings"

// pickLine is a line of the list: a step, or the heading of the role whose first shown step is row.
type pickLine struct {
	row     int
	heading bool
}

// shows says that the filter keeps r, by its role or its name.
func (l pickList) shows(r pickRow) bool {
	return strings.Contains(strings.ToLower(r.entry()), strings.ToLower(l.words))
}

// lines returns what the filter leaves of the side, each step under its role's heading. A stage
// names the steps under it, so that a group called as its stage takes no heading of its own.
func (l pickList) lines() []pickLine {
	var lines []pickLine
	parent := ""
	for i, r := range l.rows {
		if r.side != l.side || !l.shows(r) {
			continue
		}
		if r.parent != "" && r.parent != parent {
			lines = append(lines, pickLine{row: i, heading: true})
		}
		parent = r.parent
		if r.number > 0 {
			parent = r.name
		}
		lines = append(lines, pickLine{row: i})
	}
	return lines
}

// current returns the line under the cursor. A filter can leave none.
func (l pickList) current(at cursor) (pickLine, bool) {
	lines := l.lines()
	if int(at) >= len(lines) {
		return pickLine{}, false
	}
	return lines[at], true
}

// filtered returns the list under words.
func (l pickList) filtered(words string) pickList {
	l.words = words
	return l
}

// narrowed returns the list under words, and the cursor on its line while that shows.
func (l pickList) narrowed(at cursor, words string) (pickList, cursor) {
	was := l.lines()
	l = l.filtered(words)
	return l, kept(at, was, l.lines())
}

// groupBox draws a role's box, which follows its steps: "[-]" says that some are on.
func (l pickList) groupBox(parent string) string {
	on, all := l.count(func(r pickRow) bool { return r.parent == parent })
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
		return r.headingIndent() + l.groupBox(r.parent) + " " + theme.PaneTitle.Render(r.parent)
	}
	mark, style := r.state.look(theme)
	return r.indent() + l.box(r) + " " + style.Render(mark) + " " + r.name
}

// view draws height lines of the list, the cursor's gutter where shown. The list runs past a window,
// so that the cursor stays in sight.
func (l pickList) view(theme Theme, height int, at cursor, shown bool) string {
	lines := l.lines()
	if len(lines) == 0 {
		return theme.Muted.Render("No step matches.")
	}
	top := at.top(height, len(lines))
	drawn := make([]string, 0, height)
	for i := top; i < min(top+height, len(lines)); i++ {
		gutter := "  "
		if shown && i == int(at) {
			gutter = "▸ "
		}
		drawn = append(drawn, gutter+l.render(theme, lines[i]))
	}
	return strings.Join(drawn, "\n")
}
