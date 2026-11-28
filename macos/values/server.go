package values

import (
	"errors"
	"fmt"
	"io/fs"
	"path"

	"github.com/zadykian/ws/engine/core/effect"
)

// FirstServer is the name of the one server that ws sets up for now.
const FirstServer = "ws"

// ServerPath returns the path of the server called name's file, below dir, the directory that
// holds servers/.
func ServerPath(dir, name string) string {
	return path.Join(dir, "servers", name, "values.yml")
}

// LoadLines reads the server's file at file. It checks each line alone, as Lines does, so a pair of
// keys may stand apart.
func LoadLines(m effect.Machine, file string) (Conf, error) { return load(m, file, Lines) }

// load reads the file at file, which parse checks.
func load(m effect.Machine, file string, parse func(string) (Conf, []Error)) (Conf, error) {
	text, err := m.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return Conf{}, fmt.Errorf("%s is missing: macos/values/README.md says what goes in it", file)
	}
	if err != nil {
		return Conf{}, err
	}
	c, errs := parse(string(text))
	var all []error
	for _, e := range errs {
		all = append(all, errors.New(e.In(file)))
	}
	return c, errors.Join(all...)
}
