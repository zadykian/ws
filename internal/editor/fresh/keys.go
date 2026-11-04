package fresh

import "github.com/zadykian/ws/internal/editor"

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

// spell writes a stroke as Fresh does.
func spell(s editor.Stroke) press {
	key, found := names[s.Key]
	if !found {
		key = s.Key
		if len(key) > 1 {
			key = "F" + key[1:]
		}
	}
	return press{key, modifiers(s, s.Shift)}
}
