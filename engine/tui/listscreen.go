package tui

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// The places of the list's buttons.
const (
	pickTogglePlace = iota
	pickAllPlace
	pickNonePlace
	pickInvertPlace
	pickPendingPlace
	pickFilterPlace
	pickRunPlace
	pickCheckPlace
	pickQuitPlace
)

// scannedMsg ends the scan. problem says why it stopped short, if it did.
type scannedMsg struct{ problem string }

// listScreen is what both screens of lists share: the frame, the scan's end, the filter's typing
// and the leave.
type listScreen struct {
	title  string
	theme  widget.Theme
	window widget.Size
	// scanning says that events of the scan are still on their way.
	scanning bool
	// problem is why the scan stopped short.
	problem string
	// filter holds the words that the list follows.
	filter widget.Filter
	// left says that the choice has gone, and the screen waits for the work to end it.
	left bool
}

// scanned ends the scan, and keeps why it stopped short, if it did.
func (s listScreen) scanned(msg scannedMsg) listScreen {
	s.scanning, s.problem = false, msg.problem
	return s
}

// typed gives the filter a key or a paste, and returns its words and whether they differ from was.
func (s listScreen) typed(msg tea.Msg, was string) (listScreen, string, bool) {
	s.filter = s.filter.Edited(msg)
	words := s.filter.Words()
	return s, words, words != was
}

// held says that no button takes a key: the screen is leaving, or the filter takes it.
func (s listScreen) held() bool { return s.left || s.filter.Editing }

// status is the title's right side: the scan's progress, then what is chosen. Its mark stands for
// the warning of a role passed over, which a short window has no pane for.
func (s listScreen) status(on, scanned, all int, warned bool) string {
	switch {
	case s.left:
		return "leaving"
	case s.scanning:
		return fmt.Sprintf("scanning %d/%d", scanned, all)
	case warned:
		return fmt.Sprintf("! %d of %d chosen", on, all)
	}
	return fmt.Sprintf("%d of %d chosen", on, all)
}

// filtering says that the filter's line shows, over the list.
func (s listScreen) filtering() bool { return s.filter.Shows() }

// listButtons are the buttons of both screens of lists, in the order of their places.
func listButtons() []key.Binding {
	button := func(k, label string) key.Binding { return key.NewBinding(key.WithKeys(k), key.WithHelp(k, label)) }
	return []key.Binding{button("space", "toggle"), button("a", "all"), button("n", "none"), button("i", "invert"),
		button("p", "pending"), button("/", "filter"), button("enter", "run"), button("c", "check"), widget.Keys.Quit}
}
