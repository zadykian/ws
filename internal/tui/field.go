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

// boxWidth is the most columns of a box drawn over the screen.
const boxWidth = 64

// answerButton is the field's first button. Enter presses it from the field too.
var answerButton = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "answer"))

// field is the box that takes the answer to a prompt.
type field struct {
	prompt  prompt
	input   FormField
	footer  Footer
	problem string
}

// newField returns the field of p, which takes the keys at once.
func newField(p prompt, window Size) field {
	kind := TextField
	if p.hidden {
		kind = PasswordField
	}
	input := NewFormField(FieldSpec{Kind: kind, Default: p.fallback}).Focus()
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
	answer := f.input.Value()
	if f.prompt.valid != nil {
		if err := f.prompt.valid(answer); err != nil {
			f.problem = err.Error()
			return f, "", false
		}
	}
	return f, answer, true
}

// render draws the box for a window, cut to its height.
func (f field) render(theme Theme, window Size) string {
	width := min(window.Width, boxWidth)
	text := f.prompt.text
	if f.prompt.fallback != "" {
		text += " [" + f.prompt.fallback + "]"
	}
	lines := []string{lipgloss.NewStyle().Width(width - 2).Render(text), f.input.input.View()}
	if f.problem != "" {
		lines = append(lines, theme.Bad.Width(width-2).Render(f.problem))
	}
	lines = append(lines, "", f.footer.render(theme, width-2))
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
