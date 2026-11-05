package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// View draws the tabs.
func (s StateScreen) View() tea.View {
	return s.layout().View(s.theme, s.window.Width, s.window.Height)
}

// layout is the screen with its panes empty: their sizes are what their bodies are drawn for.
func (s StateScreen) layout() Layout {
	status := ""
	if s.reading {
		status = "reading"
	}
	return Layout{
		Title:  s.title,
		Status: status,
		Header: s.header(),
		Panes:  []Pane{{Title: tabNames[s.tab], Focused: true}},
		Footer: s.footer,
	}
}

// header draws the tabs, in as many lines as the window needs so that none is cut, then why the last
// read failed and what it warned of.
func (s StateScreen) header() string {
	var lines []string
	line := ""
	for tab := range tabCount {
		cell := s.tabCell(tab)
		switch {
		case line == "":
			line = cell
		case lipgloss.Width(line)+buttonGap+lipgloss.Width(cell) > s.window.Width:
			lines = append(lines, line)
			line = cell
		default:
			line += strings.Repeat(" ", buttonGap) + cell
		}
	}
	lines = append(lines, line)
	if s.problem != "" {
		lines = append(lines, s.theme.Bad.Render(s.problem))
	}
	if s.warning != "" {
		lines = append(lines, s.theme.Warn.Render(s.warning))
	}
	return strings.Join(lines, "\n")
}

// tabCell draws a tab with its count. What needs a look shows in color, and the count says the same
// under NO_COLOR. Before the first report a tab has no count to give.
func (s StateScreen) tabCell(tab int) string {
	text := " " + tabNames[tab] + " "
	if s.reported {
		text = fmt.Sprintf(" %s %d ", tabNames[tab], len(s.rows[tab]))
	}
	switch {
	case tab == s.tab:
		return s.theme.ButtonFocus.Render(text)
	case tab == conflictsTab && len(s.rows[tab]) > 0:
		return s.theme.Bad.Render(text)
	case tab == pendingTab && len(s.rows[tab]) > 0:
		return s.theme.Warn.Render(text)
	}
	return text
}
