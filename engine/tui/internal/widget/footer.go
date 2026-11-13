package widget

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ButtonGap is the columns between two buttons.
const ButtonGap = 2

// Footer is a screen's buttons. A button is a key binding: its help gives the shortcut and the
// label. A disabled one shows, and takes neither the focus nor its key. Tab walks a ring of the
// panes and the buttons, so a screen that lets Tab move between its panes takes the key first.
type Footer struct {
	buttons []key.Binding
	// Focus is the place of the focused button plus one, so that the zero Footer leaves the focus
	// with the panes.
	Focus int
}

// NewFooter returns a footer of buttons, with the focus on the panes.
func NewFooter(buttons ...key.Binding) Footer {
	return Footer{buttons: buttons}
}

// Focused returns the place of the focused button, or -1 where the panes hold the focus.
func (f Footer) Focused() int {
	return f.Focus - 1
}

// Enable turns the button at place on or off. Turned off, it hands its focus to the next in the
// ring. The buttons are copied, as a screen's older value shares them.
func (f Footer) Enable(place int, on bool) Footer {
	f.buttons = slices.Clone(f.buttons)
	f.buttons[place].SetEnabled(on)
	if !on && place == f.Focused() {
		return f.Move(1)
	}
	return f
}

// Update moves the focus on Tab and Shift-Tab. It returns the place of the button that msg
// pressed, by Enter on the focused one or by its shortcut, or -1.
func (f Footer) Update(msg tea.KeyPressMsg) (Footer, int) {
	switch {
	case key.Matches(msg, Keys.Next):
		return f.Move(1), -1
	case key.Matches(msg, Keys.Prev):
		return f.Move(-1), -1
	case f.Focus > 0 && key.Matches(msg, Keys.Press):
		return f, f.Focused()
	}
	for i, button := range f.buttons {
		if key.Matches(msg, button) {
			return f, i
		}
	}
	return f, -1
}

// Move steps the focus along the ring, past the disabled buttons. The panes' place ends the walk.
func (f Footer) Move(step int) Footer {
	ring := len(f.buttons) + 1
	for {
		f.Focus = (f.Focus + step + ring) % ring
		if f.Focus == 0 || f.buttons[f.Focus-1].Enabled() {
			return f
		}
	}
}

// rows breaks the buttons into rows of at most width columns, each a list of their places.
func (f Footer) rows(width int) [][]int {
	var rows [][]int
	used := 0
	for i, button := range f.buttons {
		size := lipgloss.Width(buttonText(button))
		if len(rows) == 0 || used+ButtonGap+size > width {
			rows = append(rows, nil)
			used = -ButtonGap
		}
		last := len(rows) - 1
		rows[last] = append(rows[last], i)
		used += ButtonGap + size
	}
	return rows
}

// Render draws the rows of buttons, or nothing where the footer has none.
func (f Footer) Render(theme Theme, width int) string {
	var lines []string
	for _, row := range f.rows(width) {
		cells := make([]string, 0, len(row))
		for _, place := range row {
			cells = append(cells, f.renderButton(theme, place))
		}
		lines = append(lines, strings.Join(cells, strings.Repeat(" ", ButtonGap)))
	}
	return strings.Join(lines, "\n")
}

// renderButton draws one button. Only a button at rest colors its shortcut apart from its label.
func (f Footer) renderButton(theme Theme, place int) string {
	button := f.buttons[place]
	switch {
	case !button.Enabled():
		return theme.ButtonOff.Render(buttonText(button))
	case place == f.Focused():
		return theme.ButtonFocus.Render(buttonText(button))
	}
	help := button.Help()
	return theme.Shortcut.Render("["+help.Key+"]") + " " + help.Desc
}

// buttonText is a button's words. The brackets tell the shortcut apart under NO_COLOR.
func buttonText(button key.Binding) string {
	help := button.Help()
	return "[" + help.Key + "] " + help.Desc
}
