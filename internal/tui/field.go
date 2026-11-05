package tui

import (
	"cmp"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
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
	input   textinput.Model
	footer  Footer
	problem string
	// secret is a hidden answer. The input never holds it: its sanitizer makes a space of a tab or
	// a line end, which an answer that shows nothing would carry unseen.
	secret string
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
	f := field{prompt: p, input: input, footer: NewFooter(answerButton, Keys.Interrupt)}
	return f.fit(window)
}

// fit sets the columns of the input, past which it scrolls.
func (f field) fit(window Size) field {
	f.input.SetWidth(max(min(window.Width, boxWidth)-2-lipgloss.Width(f.input.Prompt)-1, 1))
	return f
}

// update hands msg to the buttons, then to the input while the input holds the focus. It returns
// the place of the button that msg pressed, or -1. The input's Cmd goes: it starts a blink, or
// reads a clipboard that a server lacks, into a message that would reach the screen. A terminal's
// own paste comes as tea.PasteMsg.
func (f field) update(msg tea.Msg) (field, int) {
	if press, ok := msg.(tea.KeyPressMsg); ok {
		var place int
		f.footer, place = f.footer.Update(press)
		if place >= 0 || f.footer.Focused() >= 0 || key.Matches(press, Keys.Next, Keys.Prev) {
			return f, place
		}
	}
	if f.prompt.hidden {
		return f.edit(msg), -1
	}
	f.input, _ = f.input.Update(msg)
	return f, -1
}

// edit takes a key or a paste into a hidden answer, as a terminal's line takes them for the plain
// prompt: Backspace and Ctrl-U erase, and a paste gives its first line. Enter alone answers, as
// nothing shows what a paste brought.
func (f field) edit(msg tea.Msg) field {
	switch msg := msg.(type) {
	case tea.PasteMsg:
		line, _, _ := strings.Cut(strings.ReplaceAll(msg.Content, "\r", "\n"), "\n")
		f.secret += line
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, f.input.KeyMap.DeleteBeforeCursor):
			f.secret = ""
		case key.Matches(msg, f.input.KeyMap.DeleteCharacterBackward):
			_, size := utf8.DecodeLastRuneInString(f.secret)
			f.secret = f.secret[:len(f.secret)-size]
		default:
			f.secret += msg.Text
		}
	}
	return f
}

// answer returns what the field holds, as effect.Terminal reads it: a hidden line as typed, and
// any other trimmed, an empty one taking the default. Where Valid refuses that, it returns false,
// and the field shows why and keeps the text.
func (f field) answer() (field, string, bool) {
	if f.prompt.hidden {
		return f, f.secret, true
	}
	answer := cmp.Or(strings.TrimSpace(f.input.Value()), f.prompt.fallback)
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
	lines := []string{lipgloss.NewStyle().Width(width - 2).Render(text), f.input.View()}
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
