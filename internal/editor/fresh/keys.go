package fresh

import (
	"slices"

	"github.com/zadykian/ws/internal/editor"
)

// press is a stroke as Fresh's config spells it.
type press struct {
	Key       string   `json:"key"`
	Modifiers []string `json:"modifiers,omitempty"`
}

// names gives the keys that Fresh spells otherwise. A letter, a digit and a sign stay as they are.
var names = map[string]string{
	"up": "Up", "down": "Down", "left": "Left", "right": "Right", "home": "Home", "end": "End",
	"pageup": "PageUp", "pagedown": "PageDown", "enter": "Enter", "escape": "Esc", "tab": "Tab",
	"space": "Space", "backspace": "Backspace", "delete": "Delete", "numpad*": "*",
}

// shifted gives the sign that tmux reports for a key with Shift, with no Shift. With no tmux, Fresh
// reads the key and Shift, so a stroke with Shift and a sign takes a binding for each.
var shifted = map[string]string{
	"[": "{", "]": "}", "/": "?", "\\": "|", "-": "_", "=": "+", ",": "<", ".": ">", ";": ":",
	"'": "\"", "`": "~", "0": ")", "1": "!", "2": "@", "3": "#", "4": "$", "5": "%", "6": "^",
	"7": "&", "8": "*", "9": "(",
}

// arrivals gives the keys that reach Fresh through tmux as another key that F12 leaves free, by
// Ghostty's legacy bytes. Each takes a binding under that key as well.
var arrivals = map[string]string{
	"ctrl+alt+4":       "ctrl+alt+\\",
	"ctrl+alt+/":       "ctrl+alt+shift+-",
	"ctrl+k ctrl+/":    "ctrl+k ctrl+shift+-",
	"ctrl+shift+space": "ctrl+space",
	"alt+shift+space":  "alt+space",
}

// skipped are F12's keys that take no binding: through tmux both arrive as ctrl+h, and with no
// tmux Fresh's default deletes a word, as F12's Ctrl+Backspace does.
var skipped = []string{"ctrl+backspace", "ctrl+shift+backspace"}

func modifiers(s editor.Stroke, shift bool) []string {
	var mods []string
	for _, m := range []struct {
		name string
		on   bool
	}{{"ctrl", s.Ctrl}, {"alt", s.Alt}, {"shift", shift}, {"super", s.Cmd}} {
		if m.on {
			mods = append(mods, m.name)
		}
	}
	return mods
}

// forms spells a stroke each way that it reaches Fresh.
func forms(s editor.Stroke) []press {
	key, found := names[s.Key]
	if !found {
		key = s.Key
		if len(key) > 1 {
			key = "F" + key[1:]
		}
	}
	plain := press{key, modifiers(s, s.Shift)}
	if sign, found := shifted[s.Key]; found && s.Shift {
		return []press{{sign, modifiers(s, false)}, plain}
	}
	return []press{plain}
}

// spell gives every sequence of presses that a chord reaches Fresh by.
func spell(chord editor.Chord) ([][]press, error) {
	if slices.Contains(skipped, chord.String()) {
		return nil, nil
	}
	found := expand(chord)
	if text, arrives := arrivals[chord.String()]; arrives {
		other, err := editor.ParseChord(text)
		if err != nil {
			return nil, err
		}
		found = append(found, expand(other)...)
	}
	return found, nil
}

func expand(chord editor.Chord) [][]press {
	sequences := [][]press{nil}
	for _, s := range chord {
		var next [][]press
		for _, sequence := range sequences {
			for _, p := range forms(s) {
				next = append(next, append(sequence[:len(sequence):len(sequence)], p))
			}
		}
		sequences = next
	}
	return sequences
}
