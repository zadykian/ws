package tui

import tea "charm.land/bubbletea/v2"

// pickScreen is the list that a run is chosen from.
type pickScreen struct {
	title  string
	theme  Theme
	list   pickList
	footer Footer
	window Size
}

// newPickScreen returns the screen of groups.
func newPickScreen(title string, theme Theme, groups []Group) pickScreen {
	return pickScreen{title: title, theme: theme, list: newPickList(groups)}
}

// Init starts nothing, as what fills the list is the work's.
func (s pickScreen) Init() tea.Cmd {
	return nil
}

// Update takes the window's size.
func (s pickScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if size, ok := msg.(tea.WindowSizeMsg); ok {
		s.window = Size{size.Width, size.Height}
	}
	return s, nil
}

// layout is the screen with its panes empty, as what fills them depends on their sizes.
func (s pickScreen) layout() Layout {
	return Layout{
		Title:  s.title,
		Panes:  []Pane{{Title: "Steps", Focused: true}},
		Footer: s.footer,
	}
}

// listHeight is the lines of the pane of steps.
func (s pickScreen) listHeight() int {
	return s.layout().PaneSizes(s.window.Width, s.window.Height)[0].Height
}

// View draws the screen.
func (s pickScreen) View() tea.View {
	l := s.layout()
	l.Panes[0].Body = s.list.view(s.theme, s.listHeight())
	return l.View(s.theme, s.window.Width, s.window.Height)
}
