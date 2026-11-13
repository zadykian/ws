package conf

import (
	"errors"

	"github.com/zadykian/ws/engine/core/effect"
)

// Read reads and parses mac.conf at path, and fails on its first error.
func Read(m effect.Machine, path string) (Conf, error) {
	text, err := m.ReadFile(path)
	if err != nil {
		return Conf{}, err
	}
	c, errs := Parse(string(text))
	if len(errs) > 0 {
		return c, errors.New(errs[0].In(path))
	}
	return c, nil
}
