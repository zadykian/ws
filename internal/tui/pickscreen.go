package tui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/step"
)

// The places of the list's buttons.
const (
	pickTogglePlace = iota
	pickAllPlace
	pickNonePlace
	pickInvertPlace
	pickPendingPlace
	pickRunPlace
	pickCheckPlace
	pickQuitPlace
)

// Choice is what the list leaves to a run. No steps says that nothing runs.
type Choice struct {
	// Only names the steps as Options.Only does.
	Only []string
	// Check plans them and applies nothing.
	Check bool
}

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
	// picks takes the choice, and holds one, so that sending it waits for nobody.
	picks chan<- Choice
	// left says that the choice has gone, and the screen waits for the work to end it.
	left bool
}

// newPickScreen returns the screen of groups, its scan under way.
func newPickScreen(title string, theme Theme, groups []Group, picks chan<- Choice) pickScreen {
	button := func(k, label string) key.Binding { return key.NewBinding(key.WithKeys(k), key.WithHelp(k, label)) }
	footer := NewFooter(button("space", "toggle"), button("a", "all"), button("n", "none"), button("i", "invert"),
		button("p", "pending"), button("enter", "run"), button("c", "check"), Keys.Quit)
	s := pickScreen{title: title, theme: theme, list: newPickList(groups), footer: footer, scanning: true, picks: picks}
	return s.enabled()
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
		var leaving tea.Cmd
		s, leaving = s.pressed(msg)
		return s.enabled(), leaving
	}
	return s.enabled(), nil
}

// pressed turns the boxes that msg's button names, moves the cursor, or gives the choice.
func (s pickScreen) pressed(msg tea.KeyPressMsg) (pickScreen, tea.Cmd) {
	if s.left {
		return s, nil
	}
	var place int
	s.footer, place = s.footer.Update(msg)
	switch place {
	case pickTogglePlace:
		s.list = s.list.toggled()
	case pickAllPlace:
		s.list = s.list.chose(func(pickRow) bool { return true })
	case pickNonePlace:
		s.list = s.list.chose(func(pickRow) bool { return false })
	case pickInvertPlace:
		s.list = s.list.chose(func(r pickRow) bool { return !r.on() })
	case pickPendingPlace:
		s.list = s.list.chose(pickRow.pending)
	case pickRunPlace:
		return s.leave(Choice{Only: s.list.only()})
	case pickCheckPlace:
		return s.leave(Choice{Only: s.list.only(), Check: true})
	case pickQuitPlace:
		return s.leave(Choice{})
	default:
		s.list = s.list.moved(msg, max(s.listHeight(), 1))
	}
	return s, nil
}

// leave gives c to the work. A session quits with its work alone, and a screen never does.
func (s pickScreen) leave(c Choice) (pickScreen, tea.Cmd) {
	s.left = true
	return s, func() tea.Msg {
		s.picks <- c
		return nil
	}
}

// enabled returns s with its boxes held while the scan ticks them, as a press would race it. A run
// waits for something chosen, as a run with no Only runs every step.
func (s pickScreen) enabled() pickScreen {
	for place := pickTogglePlace; place <= pickPendingPlace; place++ {
		s.footer = s.footer.Enable(place, !s.scanning)
	}
	some := !s.scanning && len(s.list.only()) > 0
	s.footer = s.footer.Enable(pickRunPlace, some).Enable(pickCheckPlace, some)
	return s
}

// status is the title's right side: how far the scan is, then how much is chosen.
func (s pickScreen) status() string {
	on, all := s.list.count(func(pickRow) bool { return true })
	if s.left {
		return "leaving"
	}
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
