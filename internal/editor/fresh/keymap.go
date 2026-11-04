package fresh

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zadykian/ws/internal/editor"
)

// binding is an entry of keybinding_maps: a single press in Key and Modifiers, a chord in Keys.
type binding struct {
	Key       string          `json:"key,omitempty"`
	Modifiers []string        `json:"modifiers,omitempty"`
	Keys      []press         `json:"keys,omitempty"`
	Action    string          `json:"action"`
	Args      json.RawMessage `json:"args,omitempty"`
	When      string          `json:"when"`
}

// contexts gives each scope that has a target Fresh's context. A global key is bound in normal, as a
// global entry loses to the default map's normal one. Fresh passes only its UI actions on from normal
// to the explorer and the panels: menus, quick open, splits, tabs, the terminal's and the explorer's
// toggles. Save, search, replace, revert and navigate stay in the editor.
var contexts = map[editor.Scope]string{
	editor.Global: "normal", editor.Text: "normal", editor.Code: "normal",
	editor.Tree: "file_explorer", editor.Panel: "file_explorer", editor.Terminal: "terminal",
}

// shellKeys are the keys whose action Fresh's terminal yields to, F12's and the default map's. Each
// takes noop there, which ends the lookup and leaves the key to the shell. The list leaves out
// Fresh's own ways out of its terminal: ctrl+space and ctrl+] (terminal_escape), F9 (keyboard
// capture) and alt+o (the dock's focus).
var shellKeys = []string{
	"ctrl+q", "ctrl+,", "ctrl+alt+]", "ctrl+alt+[", "alt+x", "ctrl+alt+s", "alt+v", "alt+left",
	"alt+right",
	"alt+e", "alt+f", "alt+g", "alt+h", "alt+s", "alt+w", "alt+r", "alt+j", "alt+/", "alt+[",
	"alt+]", "alt+`", "ctrl+p", "ctrl+;", "ctrl+'",
}

// bindings renders the keymap's keys, in the order of its scopes, and the terminal's noops.
func bindings(keymap editor.Keymap, targets map[string]editor.Target) ([]binding, error) {
	var out []binding
	var problems []error
	for _, scope := range editor.Scopes {
		for _, b := range keymap[scope] {
			target := targets[b.Action]
			if target.To == "" {
				continue
			}
			when, found := contexts[scope]
			if !found {
				problems = append(problems, fmt.Errorf("%s: no context for %s", b.Action, scope))
				continue
			}
			for _, chord := range b.Keys {
				rendered, err := entries(chord, target, when)
				if err != nil {
					return nil, err
				}
				out = append(out, rendered...)
			}
		}
	}
	for _, key := range shellKeys {
		chord, err := editor.ParseChord(key)
		if err != nil {
			return nil, err
		}
		rendered, err := entries(chord, editor.Target{To: "noop"}, "terminal")
		if err != nil {
			return nil, err
		}
		out = append(out, rendered...)
	}
	return out, errors.Join(problems...)
}

func entries(chord editor.Chord, target editor.Target, when string) ([]binding, error) {
	sequences, err := spell(chord)
	if err != nil {
		return nil, err
	}
	out := make([]binding, len(sequences))
	for i, sequence := range sequences {
		out[i] = binding{Action: target.To, Args: json.RawMessage(target.Args), When: when}
		if len(sequence) == 1 {
			out[i].Key, out[i].Modifiers = sequence[0].Key, sequence[0].Modifiers
		} else {
			out[i].Keys = sequence
		}
	}
	return out, nil
}
