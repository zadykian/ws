package tui

import (
	tea "charm.land/bubbletea/v2"
)

// View draws the list.
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
		Panes:  []Pane{{Title: tabNames[s.tab], Focused: true}},
	}
}
