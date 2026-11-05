package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

var window = Size{80, 24}

// typeIn gives the field each rune of text as a key.
func typeIn(f field, text string) field {
	for _, r := range text {
		f = f.update(press(r))
	}
	return f
}

func TestFieldKeys(t *testing.T) {
	f := newField(prompt{text: "Host"}, window)
	f = f.update(tea.PasteMsg{Content: "pasted"})
	if got := typeIn(f, "!").input.Value(); got != "pasted!" {
		t.Errorf("a paste and a key: %q", got)
	}
}
