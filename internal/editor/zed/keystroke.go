package zed

import (
	"strings"

	"github.com/zadykian/ws/internal/editor"
)

// shifted gives the sign a Mac reports for a key with Shift, without the Shift.
var shifted = map[string]string{
	"[": "{", "]": "}", "/": "?", "\\": "|", "-": "_", "=": "+", ",": "<", ".": ">", ";": ":",
	"'": "\"", "`": "~", "0": ")", "1": "!", "2": "@", "3": "#", "4": "$", "5": "%", "6": "^",
	"7": "&", "8": "*", "9": "(",
}

// keystroke spells a chord in Zed's terms: its keys joined by spaces, each with its modifiers.
func keystroke(chord editor.Chord) string {
	parts := make([]string, len(chord))
	for i, s := range chord {
		parts[i] = stroke(s)
	}
	return strings.Join(parts, " ")
}

func stroke(s editor.Stroke) string {
	key, shift := s.Key, s.Shift
	if sign, found := shifted[key]; found && shift {
		key, shift = sign, false
	}
	if key == "numpad*" {
		key = "*"
	}
	var mods []string
	for _, m := range []struct {
		name string
		on   bool
	}{{"ctrl", s.Ctrl}, {"alt", s.Alt}, {"cmd", s.Cmd}, {"shift", shift}} {
		if m.on {
			mods = append(mods, m.name)
		}
	}
	return strings.Join(append(mods, key), "-")
}
