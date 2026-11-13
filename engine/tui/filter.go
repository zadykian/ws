package tui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// filterPrompt starts the filter's line, with the key that opens it.
const filterPrompt = "/ "

// ClearKey drops the filter.
var ClearKey = key.NewBinding(key.WithKeys("esc"))

// Filter is the line of words that narrows a list. textinput holds the words, so that its editing
// keys and a paste work on the lists and the state screen alike.
type Filter struct {
	input textinput.Model
	// Editing says that the keys go to the line.
	Editing bool
}

// NewFilter is a line of no words, not yet taking keys.
func NewFilter() Filter {
	input := newInput()
	input.Prompt = filterPrompt
	return Filter{input: input}
}

// Words returns what the line holds.
func (f Filter) Words() string {
	return f.input.Value()
}

// Shows says that the line takes a place over the list: while it has the keys, or its words stand.
func (f Filter) Shows() bool {
	return f.Editing || f.Words() != ""
}

// Opened gives the line the keys.
func (f Filter) Opened() Filter {
	f.Editing = true
	return f
}

// Cleared drops the words.
func (f Filter) Cleared() Filter {
	f.input.Reset()
	return f
}

// Edited gives the line a key or a paste. Enter keeps the words and Esc drops them, and each gives
// the keys back. The input's Cmd goes, as a field's does.
func (f Filter) Edited(msg tea.Msg) Filter {
	if press, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(press, Keys.Press):
			f.Editing = false
			return f
		case key.Matches(press, ClearKey):
			f.Editing = false
			return f.Cleared()
		}
	}
	f.input, _ = f.input.Update(msg)
	return f
}

// Line draws the filter: the input while it has the keys, the words once they stand, and nothing
// without any.
func (f Filter) Line(theme Theme) string {
	switch {
	case f.Editing:
		return f.input.View()
	case f.Words() != "":
		return theme.Muted.Render(filterPrompt + f.Words())
	}
	return ""
}
