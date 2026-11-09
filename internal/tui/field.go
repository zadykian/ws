package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// The places of a field's buttons.
const (
	answerPlace = iota
	stopPlace
)

const (
	// boxWidth is the most columns of a box drawn over the screen.
	boxWidth = 64
	// minQuestionHeight is the fewest rows that frame a line of a question over its input.
	minQuestionHeight = minPaneHeight + 1
)

// answerButton is the field's first button. Enter presses it from the field too.
var answerButton = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "answer"))

// field is the box that takes the answer to a prompt.
type field struct {
	prompt prompt
	input  FormField
	footer Footer
}

// newField returns the field of p, which takes the keys at once.
func newField(p prompt, window Size) field {
	kind := TextField
	switch {
	case p.hidden:
		kind = PasswordField
	case len(p.options) > 0:
		kind = ChoiceField
	}
	spec := FieldSpec{Kind: kind, Label: p.text, Default: p.fallback, Options: p.options, Valid: p.valid}
	input := NewFormField(spec).Focus()
	f := field{prompt: p, input: input, footer: NewFooter(answerButton, Keys.Interrupt)}
	return f.fit(window)
}

// fit sets the columns of the input, past which it scrolls.
func (f field) fit(window Size) field {
	f.input = f.input.Fit(min(window.Width, boxWidth) - 2)
	return f
}

// update hands msg to the buttons, then to the input while the input holds the focus. It returns
// the place of the button that msg pressed, or -1.
func (f field) update(msg tea.Msg) (field, int) {
	if press, ok := msg.(tea.KeyPressMsg); ok {
		var place int
		f.footer, place = f.footer.Update(press)
		if place >= 0 || f.footer.Focused() >= 0 || key.Matches(press, Keys.Next, Keys.Prev) {
			return f, place
		}
	}
	f.input = f.input.Update(msg)
	return f, -1
}

// answer returns what the field holds. Where Valid refuses that, it returns false, and the field
// shows why and keeps the text.
func (f field) answer() (field, string, bool) {
	var ok bool
	f.input, ok = f.input.Check()
	return f, f.input.Value(), ok
}

// render draws the box for a window. The buttons go first in a short window, as their keys work
// unseen, and then the frame. A run waits on the question, and its keys reach the input, so the
// input stays. A refusal takes the line under the input, so that the box holds still as one comes
// and goes.
func (f field) render(theme Theme, window Size) string {
	if window.Width < 1 {
		return ""
	}
	if window.Width < minFrameWidth || window.Height < minQuestionHeight {
		lines := f.input.Lines(theme, window.Width, window.Height)
		return lipgloss.NewStyle().MaxWidth(window.Width).Render(strings.Join(lines, "\n"))
	}
	width, room := min(window.Width, boxWidth)-2, window.Height-2
	lines := f.input.Lines(theme, width, room)
	footer := f.footer.render(theme, width)
	spare := room - len(lines) - lipgloss.Height(footer)
	if spare > 0 && f.input.problem == nil {
		lines = append(lines, "")
	}
	if spare >= 0 {
		lines = append(lines, footer)
	}
	return box(theme, "Question", strings.Join(lines, "\n"), window)
}

// box draws body in a focused frame that fits it, or nothing where the window holds no frame.
func box(theme Theme, title, body string, window Size) string {
	if window.Width < minFrameWidth || window.Height < minPaneHeight {
		return ""
	}
	frame := Size{min(window.Width, boxWidth), min(lipgloss.Height(body)+2, window.Height)}
	return Pane{Title: title, Body: body, Focused: true}.render(theme, frame)
}
