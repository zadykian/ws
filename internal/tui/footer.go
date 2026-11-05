package tui

import (
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

// Update moves the focus on Tab and Shift-Tab.
func (f Footer) Update(msg tea.KeyPressMsg) Footer {
	switch {
	case key.Matches(msg, Keys.Next):
		return f.move(1)
	case key.Matches(msg, Keys.Prev):
		return f.move(-1)
	}
	return f
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
