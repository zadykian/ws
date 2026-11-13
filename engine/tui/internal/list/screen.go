package list

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// The places of the list's buttons.
const (
	PickTogglePlace = iota
	PickAllPlace
	PickNonePlace
	PickInvertPlace
	PickPendingPlace
	PickFilterPlace
	PickRunPlace
	PickCheckPlace
	PickQuitPlace
)

// ScannedMsg ends the scan. problem says why it stopped short, if it did.
type ScannedMsg struct{ Problem string }

// Screen is what both screens of lists share: the frame, the scan's end, the filter's typing
// and the leave.
type Screen struct {
	Title  string
	Theme  widget.Theme
	Window widget.Size
	// Scanning says that events of the scan are still on their way.
	Scanning bool
	// Problem is why the scan stopped short.
	Problem string
	// Filter holds the words that the list follows.
	Filter widget.Filter
	// Left says that the choice has gone, and the screen waits for the work to end it.
	Left bool
}

// Scanned ends the scan, and keeps why it stopped short, if it did.
func (s Screen) Scanned(msg ScannedMsg) Screen {
	s.Scanning, s.Problem = false, msg.Problem
	return s
}

// Typed gives the filter a key or a paste, and returns its words and whether they differ from was.
func (s Screen) Typed(msg tea.Msg, was string) (Screen, string, bool) {
	s.Filter = s.Filter.Edited(msg)
	words := s.Filter.Words()
	return s, words, words != was
}

// Held says that no button takes a key: the screen is leaving, or the filter takes it.
func (s Screen) Held() bool { return s.Left || s.Filter.Editing }

// Status is the title's right side: the scan's progress, then what is chosen. Its mark stands for
// the warning of a role passed over, which a short window has no pane for.
func (s Screen) Status(on, scanned, all int, warned bool) string {
	switch {
	case s.Left:
		return "leaving"
	case s.Scanning:
		return fmt.Sprintf("scanning %d/%d", scanned, all)
	case warned:
		return fmt.Sprintf("! %d of %d chosen", on, all)
	}
	return fmt.Sprintf("%d of %d chosen", on, all)
}

// Filtering says that the filter's line shows, over the list.
func (s Screen) Filtering() bool { return s.Filter.Shows() }

// Buttons are the buttons of both screens of lists, in the order of their places.
func Buttons() []key.Binding {
	button := func(k, label string) key.Binding { return key.NewBinding(key.WithKeys(k), key.WithHelp(k, label)) }
	return []key.Binding{button("space", "toggle"), button("a", "all"), button("n", "none"), button("i", "invert"),
		button("p", "pending"), button("/", "filter"), button("enter", "run"), button("c", "check"), widget.Keys.Quit}
}
