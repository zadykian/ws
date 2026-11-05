package tui

import (
	tea "charm.land/bubbletea/v2"
)

// session is a run's model: its screen, which takes every message that the session doesn't.
type session struct {
	screen tea.Model
	window Size
	// begin starts the work.
	begin tea.Cmd
}

// Init starts the screen and the work.
func (s session) Init() tea.Cmd {
	return tea.Batch(s.screen.Init(), s.begin)
}

// Update takes what is a run's, and hands the rest to the screen.
func (s session) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		s.window = Size{msg.Width, msg.Height}
	}
	var cmd tea.Cmd
	s.screen, cmd = s.screen.Update(msg)
	return s, cmd
}

// View draws the screen. It holds to the alternate buffer whatever the screen says, as the end of a
// run restores the terminal by it.
func (s session) View() tea.View {
	view := s.screen.View()
	view.AltScreen = true
	return view
}
