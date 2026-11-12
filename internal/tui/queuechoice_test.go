package tui

import "slices"

// atRow returns l turned to the side of the row of that name, and the cursor on the row.
func atRow(l pickList, name string) (pickList, cursor) {
	row := slices.IndexFunc(l.rows, func(r pickRow) bool { return r.name == name })
	l.side = l.rows[row].side
	return l, cursor(slices.Index(l.lines(), pickLine{row: row}))
}

// drawn is what a side of l shows, a line each, its colors gone.
func drawn(l pickList, side Side) []string {
	l.side = side
	var got []string
	for _, line := range l.lines() {
		got = append(got, stripped(l.render(testTheme, line)))
	}
	return got
}
