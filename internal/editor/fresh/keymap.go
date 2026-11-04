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

// bindings renders the keymap's keys, in the order of its scopes.
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
