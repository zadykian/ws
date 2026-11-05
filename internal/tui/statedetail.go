package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// detailWeight is the detail pane's part of the width: a diff needs more room than a name.
const detailWeight = 2

// detailLines draws the selected row for a pane of width, each line wrapped to it, as what a pane
// cuts is what a diff is read for.
func (s StateScreen) detailLines(width int) []string {
	rows := s.rows[s.tab]
	if len(rows) == 0 || width <= 0 {
		return nil
	}
	row := rows[s.cursors[s.tab]]
	var lines []string
	add := func(style lipgloss.Style, text string) {
		for line := range strings.SplitSeq(lipgloss.Wrap(text, width, ""), "\n") {
			lines = append(lines, style.Render(line))
		}
	}
	add(s.theme.PaneTitle, row.name)
	for _, fact := range row.facts {
		add(lipgloss.NewStyle(), fact.label+": "+fact.text)
	}
	if row.conflict != "" {
		add(s.theme.Bad, "Conflict: "+row.conflict)
	}
	if row.diff != "" {
		lines = append(lines, "")
		for i, line := range strings.Split(strings.TrimSuffix(row.diff, "\n"), "\n") {
			add(s.diffStyle(i, line), line)
		}
	}
	return lines
}

// diffStyle colors the line at place i of a unified diff. Its marks say the same under NO_COLOR. The
// first two lines name the file, and any later one that starts as they do is a change. A removal is
// not bold, as a long one would shout over the rest.
func (s StateScreen) diffStyle(i int, line string) lipgloss.Style {
	switch {
	case i < 2 && (strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ")):
		return s.theme.Muted
	case strings.HasPrefix(line, "@@"):
		return s.theme.Muted
	case strings.HasPrefix(line, "+"):
		return s.theme.Good
	case strings.HasPrefix(line, "-"):
		return s.theme.Bad.Bold(false)
	}
	return lipgloss.NewStyle()
}
