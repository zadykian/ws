package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

var window = Size{80, 24}

// typeIn gives the field each rune of text as a key.
func typeIn(f field, text string) field {
	for _, r := range text {
		f, _ = f.update(press(r))
	}
	return f
}

func TestFieldKeys(t *testing.T) {
	f := newField(prompt{text: "Host"}, window)
	if _, place := f.update(enter); place != answerPlace {
		t.Errorf("Enter in the input pressed %d", place)
	}
	f, _ = f.update(tab)
	f, _ = f.update(tab)
	if got := typeIn(f, "x").input.Value(); got != "" {
		t.Errorf("a button in focus let %q into the input", got)
	}
	if _, place := f.update(enter); place != stopPlace {
		t.Errorf("Enter on the second button pressed %d", place)
	}
	f, _ = f.update(tab)
	f, _ = f.update(tea.PasteMsg{Content: "pasted"})
	if got := typeIn(f, "!").input.Value(); got != "pasted!" {
		t.Errorf("Tab back to the input, a paste and a key: %q", got)
	}
}
