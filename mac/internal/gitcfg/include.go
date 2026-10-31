package gitcfg

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

// ours reports whether an include path names ~/.config/ws/gitconfig, as git config prints an include
// of ~/.gitconfig: by the home's path, or relative to the home.
func ours(path, dir string) bool {
	return path == dir+"/gitconfig" || path == ".config/ws/gitconfig"
}

// Included reports whether ~/.gitconfig includes ~/.config/ws/gitconfig. Its error says what makes
// the include wrong, after the file's name: the include comes twice or sits below the file's own
// settings, which it then overrides. paths are the file's include.path values, and first is the
// name of its first setting, as git config --name-only --list prints it.
func Included(paths []string, first, dir string) (bool, error) {
	count := 0
	for _, p := range paths {
		if ours(p, dir) {
			count++
		}
	}
	if count == 0 {
		return false, nil
	}
	if first == "include.path" {
		first = paths[0]
	}
	if !ours(first, dir) {
		return false, errors.New("includes ~/.config/ws/gitconfig below its own settings, which " +
			"it then overrides: move the include to the top")
	}
	if count != 1 {
		return false, fmt.Errorf("includes ~/.config/ws/gitconfig %d times: keep the one at the top", count)
	}
	return true, nil
}

var ownSetting = regexp.MustCompile(`^(user\.(name|email|signingkey)|commit\.gpgsign|` +
	`gpg\.(format|ssh\.program|ssh\.allowedsignersfile))$`)

// Own returns the settings of identity and signing among names, once each, in their order, joined
// by ", ". names are those that git config --name-only --list prints.
func Own(names []string) string {
	var list []string
	for _, n := range names {
		if ownSetting.MatchString(n) && !slices.Contains(list, n) {
			list = append(list, n)
		}
	}
	return strings.Join(list, ", ")
}
