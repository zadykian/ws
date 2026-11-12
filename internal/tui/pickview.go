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

// named says that the filter's words are in name: a bare group's steps stand under it without a heading.
func (l pickList) named(name string) bool {
	return strings.Contains(strings.ToLower(name), strings.ToLower(l.words))
}

// lines returns what the filter leaves of the side, each step under its role's heading. A stage
// names the steps under it, so that a group called as its stage takes no heading of its own.
func (l pickList) lines() []pickLine {
	var lines []pickLine
	parent := ""
	for i, r := range l.rows {
		if !l.shows(r) {
			continue
		}
		if r.parent != "" && r.parent != parent && (r.form != groupBare || !l.named(r.parent)) {
			lines = append(lines, pickLine{row: i, heading: true})
		}
		parent = r.parent
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
		return l.groupBox(r.parent) + " " + theme.PaneTitle.Render(r.parent)
	}
	mark, style := r.state.look(theme)
	return r.indented() + r.box() + " " + style.Render(mark) + " " + r.name
}

// view draws height lines of the list, the cursor's gutter always.
func (l pickList) view(theme Theme, height int, at cursor) string {
	lines := l.lines()
	return windowed(theme, height, at, true, len(lines), func(i int) string { return l.render(theme, lines[i]) })
}

// windowed draws height of count lines, each by draw, the cursor's gutter where shown. The lines run
// past a window, so that the cursor stays in sight.
func windowed(theme Theme, height int, at cursor, shown bool, count int, draw func(int) string) string {
	if count == 0 {
		return theme.Muted.Render("No step matches.")
	}
	top := at.top(height, count)
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
