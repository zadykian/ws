package tui

import (
	"slices"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Footer is a screen's buttons. A button is a key binding: its help gives the shortcut and the
// label. A disabled one shows, and takes neither the focus nor its key. Tab walks a ring of the
// panes and the buttons, so a screen that lets Tab move between its panes takes the key first.
type Footer struct {
	buttons []key.Binding
	// focus is the place of the focused button plus one, so that the zero Footer leaves the focus
	// with the panes.
	focus int
}

// NewFooter returns a footer of buttons, with the focus on the panes.
func NewFooter(buttons ...key.Binding) Footer {
	return Footer{buttons: buttons}
}

// Focused returns the place of the focused button, or -1 where the panes hold the focus.
func (f Footer) Focused() int {
	return f.focus - 1
}

// Enable turns the button at place on or off. Turned off, it hands its focus to the next in the
// ring. The buttons are copied, as a screen's older value shares them.
func (f Footer) Enable(place int, on bool) Footer {
	f.buttons = slices.Clone(f.buttons)
	f.buttons[place].SetEnabled(on)
	if !on && place == f.Focused() {
		return f.move(1)
	}
	return f
}

// Update moves the focus on Tab and Shift-Tab. It returns the place of the button that msg
// pressed, by Enter on the focused one or by its shortcut, or -1.
func (f Footer) Update(msg tea.KeyPressMsg) (Footer, int) {
	switch {
	case key.Matches(msg, Keys.Next):
		return f.move(1), -1
	case key.Matches(msg, Keys.Prev):
		return f.move(-1), -1
	case f.focus > 0 && key.Matches(msg, Keys.Press):
		return f, f.Focused()
	}
	for i, button := range f.buttons {
		if key.Matches(msg, button) {
			return f, i
		}
	}
	return f, -1
}

// move steps the focus along the ring, past the disabled buttons. The panes' place ends the walk.
func (f Footer) move(step int) Footer {
	ring := len(f.buttons) + 1
	for {
		f.focus = (f.focus + step + ring) % ring
		if f.focus == 0 || f.buttons[f.focus-1].Enabled() {
			return f
		}
	}
}
