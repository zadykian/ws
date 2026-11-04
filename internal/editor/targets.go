package editor

import (
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
