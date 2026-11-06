package tui

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
