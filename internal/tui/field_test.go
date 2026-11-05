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

func TestFieldAnswerReadsAsThePlainPromptDoes(t *testing.T) {
	cases := []struct {
		name   string
		prompt prompt
		typed  string
		pasted string
		want   string
	}{
		{"typed", prompt{text: "Host"}, "ws", "", "ws"},
		{"blanks trimmed", prompt{text: "Host"}, "  ws ", "", "ws"},
		{"the default", prompt{text: "Host", fallback: "ws"}, "  ", "", "ws"},
		{"a default passed over", prompt{text: "Host", fallback: "ws"}, "mac", "", "mac"},
		{"hidden blanks kept", prompt{text: "Password", hidden: true}, " s3 ", "", " s3 "},
		{"empty and hidden", prompt{text: "Password", hidden: true}, "", "", ""},
		{"a hidden paste's line end", prompt{text: "Password", hidden: true}, "s", "3\r\n", "s3"},
		{"a hidden paste's first line", prompt{text: "Password", hidden: true}, "", "s3\nx", "s3"},
		{"a hidden paste's tab", prompt{text: "Password", hidden: true}, "", "a\tb", "a\tb"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f, _ := typeIn(newField(c.prompt, window), c.typed).update(tea.PasteMsg{Content: c.pasted})
			_, got, ok := f.answer()
			if !ok || got != c.want {
				t.Errorf("answer = %q, %v; want %q", got, ok, c.want)
			}
		})
	}
}

func TestFieldErasesAHiddenAnswerAsATerminalDoes(t *testing.T) {
	f := typeIn(newField(prompt{text: "Password", hidden: true}, window), "typo")
	f, _ = f.update(tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl})
	f, _ = typeIn(f, "sé").update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	if _, got, _ := f.answer(); got != "s" {
		t.Errorf("past Ctrl-U and Backspace: %q", got)
	}
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
