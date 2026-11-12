package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// Words of the panes where no row shows.
const (
	readingNote = "Reading the machine."
	emptyNote   = "None."
)

// View draws the tabs, the list and the detail pane.
func (s StateScreen) View() tea.View {
	l := s.layout()
	sizes := l.PaneSizes(s.window.Width, s.window.Height)
	l.Panes[listPane].Body = s.listBody(sizes[listPane])
	l.Panes[detailPane].Title, l.Panes[detailPane].Body = s.detailBody(sizes[detailPane])
	return l.View(s.theme, s.window.Width, s.window.Height)
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
		Panes: []Pane{
			{Title: tabNames[s.tab], Focused: !s.inDetail},
			{Title: "Detail", Weight: detailWeight, Focused: s.inDetail},
		},
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

// tabCell draws a tab with its count, and under a filter what that leaves of it. What needs a look
// shows in color, and the count says the same under NO_COLOR. Before the first report a tab has no
// count to give.
func (s StateScreen) tabCell(tab int) string {
	text := " " + tabNames[tab] + " "
	switch {
	case s.reported && s.filter.words() != "":
		text = fmt.Sprintf(" %s %d/%d ", tabNames[tab], len(s.shown(tab)), len(s.rows[tab]))
	case s.reported:
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

// listBody draws the rows that fit a pane of size. The selection stays in the middle of a long
// list. So the list needs no scroll of its own to keep, and a new size can't lose the row.
func (s StateScreen) listBody(size Size) string {
	var lines []string
	if field := s.filter.line(s.theme); field != "" {
		lines = append(lines, field)
	}
	rows := s.shown(s.tab)
	room := size.Height - len(lines)
	cursor := s.cursors[s.tab]
	top := max(min(cursor-room/2, len(rows)-room), 0)
	for i := top; i < min(top+room, len(rows)); i++ {
		lines = append(lines, s.rowLine(rows[i], i == cursor, size.Width))
	}
	switch {
	case len(rows) > 0:
	case s.reported:
		lines = append(lines, s.theme.Muted.Render(emptyNote))
	case s.reading:
		lines = append(lines, s.theme.Muted.Render(readingNote))
	}
	return strings.Join(lines, "\n")
}

// rowLine draws a row. The selected one is reversed across the pane, which needs no color, and a
// conflict has a mark for the same reason.
func (s StateScreen) rowLine(row stateRow, selected bool, width int) string {
	text := "  " + row.name
	style := lipgloss.NewStyle()
	if row.conflict != "" {
		text = "! " + row.name
		style = s.theme.Bad
	}
	if selected {
		text += strings.Repeat(" ", max(width-lipgloss.Width(text), 0))
		style = s.theme.ButtonFocus
	}
	return style.MaxWidth(width).Render(text)
}
