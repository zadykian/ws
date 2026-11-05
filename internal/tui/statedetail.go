package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// detailWeight is the detail pane's part of the width: a diff needs more room than a name.
const detailWeight = 2

// detailLines draws the selected row for a pane of width, each line wrapped to it.
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
	return lines
}
