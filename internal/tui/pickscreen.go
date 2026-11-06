package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/step"
)

// scannedMsg ends the scan. problem says why it stopped short, if it did.
type scannedMsg struct{ problem string }

// pickScreen is the list that a run is chosen from. A scan fills it behind the screen, and the
// maintainer can read and move as it goes.
type pickScreen struct {
	title  string
	theme  Theme
	list   pickList
	footer Footer
	window Size
	// scanning says that events of the scan are still on their way.
	scanning bool
	// problem is why the scan stopped short.
	problem string
}

// newPickScreen returns the screen of groups, its scan under way.
func newPickScreen(title string, theme Theme, groups []Group) pickScreen {
	return pickScreen{title: title, theme: theme, list: newPickList(groups), scanning: true}
}

// Init starts nothing, as the scan is the work's.
func (s pickScreen) Init() tea.Cmd {
	return nil
}

// Update takes the scan's events and lines, the window's size and the keys.
func (s pickScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case step.Event:
		s.list = s.list.take(msg)
	case Line:
		s.list = s.list.hear(msg)
	case scannedMsg:
		s.scanning, s.problem = false, msg.problem
	case tea.WindowSizeMsg:
		s.window = Size{msg.Width, msg.Height}
	case tea.KeyPressMsg:
		s.list = s.list.moved(msg, max(s.listHeight(), 1))
	}
	return s, nil
}

// status is the title's right side: how far the scan is, then how much is chosen.
func (s pickScreen) status() string {
	on, all := s.list.count(func(pickRow) bool { return true })
	if s.scanning {
		_, scanned := s.list.count(func(r pickRow) bool { return r.state != stepOpen })
		return fmt.Sprintf("scanning %d/%d", scanned, all)
	}
	return fmt.Sprintf("%d of %d chosen", on, all)
}

// layout is the screen with its panes empty, as what fills them depends on their sizes.
func (s pickScreen) layout() Layout {
	return Layout{
		Title:  s.title,
		Status: s.status(),
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
