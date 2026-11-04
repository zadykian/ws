package editor

import (
	"errors"
	"fmt"
	"io/fs"
)

// Target says what an editor does with an action: its own To, with Args, or nothing, for the
// reason Why. KeyArg passes the key as the argument.
type Target struct {
	To     string `yaml:"to"`
	Args   string `yaml:"args"`
	KeyArg bool   `yaml:"key_arg"`
	Why    string `yaml:"why"`
}

// LoadTargets reads an adapter's files, each a map of action to target.
func LoadTargets(files fs.FS, pattern string) (map[string]Target, error) {
	names, err := fs.Glob(files, pattern)
	if err != nil {
		return nil, err
	}
	targets := map[string]Target{}
	for _, name := range names {
		data, err := fs.ReadFile(files, name)
		if err != nil {
			return nil, err
		}
		var file map[string]Target
		if err := decodeStrict(data, &file); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		for action, target := range file {
			if _, twice := targets[action]; twice {
				return nil, fmt.Errorf("%s: %s is there twice", name, action)
			}
			targets[action] = target
		}
	}
	return targets, nil
}

// Check holds an adapter to its contract. Each action of the keymap has a target, which either
// names the editor's action or says why it has none, and each target names an action of the keymap.
func (k Keymap) Check(targets map[string]Target) error {
	used := map[string]bool{}
	var problems []error
	for _, scope := range Scopes {
		for _, b := range k[scope] {
			if used[b.Action] {
				continue
			}
			used[b.Action] = true
			target, found := targets[b.Action]
			switch {
			case !found:
				problems = append(problems, fmt.Errorf("%s has no target and no reason", b.Action))
			case (target.To == "") == (target.Why == ""):
				problems = append(problems, fmt.Errorf("%s needs a target or a reason", b.Action))
			}
		}
	}
	for action := range targets {
		if !used[action] {
			problems = append(problems, fmt.Errorf("%s: no such action in the keymap", action))
		}
	}
	return errors.Join(problems...)
}
