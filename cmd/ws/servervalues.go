package main

import (
	"errors"
	"io/fs"
	"path"

	"github.com/zadykian/ws/effect"
)

// writeVars replaces the file. A first one is root's alone, in a directory that anyone reads.
func writeVars(m effect.Machine, data []byte) error {
	dir := path.Dir(serverVars)
	if _, err := m.Stat(dir); errors.Is(err, fs.ErrNotExist) {
		if err := m.Mkdir(dir, 0o755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	attrs, err := m.RootAttrs(serverVars)
	if errors.Is(err, fs.ErrNotExist) {
		attrs, err = effect.Attrs{Mode: 0o600}, nil
	}
	if err != nil {
		return err
	}
	return m.WriteAtomic(serverVars, data, attrs)
}
