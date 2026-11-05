package tui

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// boxWidth is the most columns of a box drawn over the screen.
const boxWidth = 64

// field is the box that takes the answer to a prompt.
type field struct {
	prompt prompt
	input  textinput.Model
}

// newField returns the field of p, which takes the keys at once. Its cursor stays lit, as a blink
// would wake the screen twice a second while a question waits. Bubbles' colors go, as NO_COLOR
// counts in the Theme alone, and a reversed cursor needs none.
func newField(p prompt, window Size) field {
	input := textinput.New()
	styles := input.Styles()
	styles.Cursor.Color = nil
	styles.Focused.Prompt = lipgloss.NewStyle()
	input.SetStyles(styles)
	_ = input.Focus()
	f := field{prompt: p, input: input}
	return f.fit(window)
}

// fit sets the columns of the input, past which it scrolls.
func (f field) fit(window Size) field {
	f.input.SetWidth(max(min(window.Width, boxWidth)-2-lipgloss.Width(f.input.Prompt)-1, 1))
	return f
}

// update hands msg to the input. The input's Cmd goes: it starts a blink, or reads a clipboard that
// a server lacks, into a message that would reach the screen. A terminal's own paste comes as
// tea.PasteMsg.
func (f field) update(msg tea.Msg) field {
	f.input, _ = f.input.Update(msg)
	return f
}
