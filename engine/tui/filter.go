package tui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// filterPrompt starts the filter's line, with the key that opens it.
const filterPrompt = "/ "

// clearKey drops the filter.
var clearKey = key.NewBinding(key.WithKeys("esc"))

// filter is the line of words that narrows a list. textinput holds the words, so that its editing
// keys and a paste work on the lists and the state screen alike.
type filter struct {
	input textinput.Model
	// editing says that the keys go to the line.
	editing bool
}

func newFilter() filter {
	input := newInput()
	input.Prompt = filterPrompt
	return filter{input: input}
}

// words returns what the line holds.
func (f filter) words() string {
	return f.input.Value()
}

// shows says that the line takes a place over the list: while it has the keys, or its words stand.
func (f filter) shows() bool {
	return f.editing || f.words() != ""
}

// opened gives the line the keys.
func (f filter) opened() filter {
	f.editing = true
	return f
}

// cleared drops the words.
func (f filter) cleared() filter {
	f.input.Reset()
	return f
}

// edited gives the line a key or a paste. Enter keeps the words and Esc drops them, and each gives
// the keys back. The input's Cmd goes, as a field's does.
func (f filter) edited(msg tea.Msg) filter {
	if press, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(press, Keys.Press):
			f.editing = false
			return f
		case key.Matches(press, clearKey):
			f.editing = false
			return f.cleared()
		}
	}
	f.input, _ = f.input.Update(msg)
	return f
}

// line draws the filter: the input while it has the keys, the words once they stand, and nothing
// without any.
func (f filter) line(theme Theme) string {
	switch {
	case f.editing:
		return f.input.View()
	case f.words() != "":
		return theme.Muted.Render(filterPrompt + f.words())
	}
	return ""
}
