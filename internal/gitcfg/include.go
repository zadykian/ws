package gitcfg

import (
	"errors"
	"fmt"

	"github.com/zadykian/ws/internal/etcws"
)

// Path is the file of ws's git settings on both sides, root's and 644, that ~/.gitconfig includes
// at its top.
const Path = etcws.Dir + "/gitconfig"

// Include is the lines at the top of ~/.gitconfig, so that the file's own settings win.
const Include = "[include]\n\tpath = " + Path + "\n"

// Included reports whether a ~/.gitconfig includes Path once, at its top. Its error says what is
// wrong, after the file's name: the include comes twice or sits below the file's own settings,
// which it then overrides. paths are the file's include.path values, and first is the name of its
// first setting, as git config --name-only --list prints it.
func Included(paths []string, first string) (bool, error) {
	count := 0
	for _, p := range paths {
		if p == Path {
			count++
		}
	}
	if count == 0 {
		return false, nil
	}
	if first == "include.path" {
		first = paths[0]
	}
	if first != Path {
		return false, errors.New("includes " + Path + " below its own settings, which " +
			"it then overrides: move the include to the top")
	}
	if count != 1 {
		return false, fmt.Errorf("includes "+Path+" %d times: keep the one at the top", count)
	}
	return true, nil
}
