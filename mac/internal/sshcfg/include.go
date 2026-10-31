package sshcfg

import (
	"regexp"
	"strings"
)

// Place says where an ssh_config file has an Include.
type Place int

// The places of an Include.
const (
	Missing Place = iota
	// First means that no setting comes before the Include.
	First
	// Below means that a setting comes before it, which ssh would take first.
	Below
)

var afterKey = regexp.MustCompile(`^[ \t]*=?[ \t]*`)

// keyAndRest splits a trimmed line at its key.
func keyAndRest(line string) (key, rest string) {
	i := strings.IndexAny(line, " \t=")
	if i < 0 {
		return line, ""
	}
	return line[:i], afterKey.ReplaceAllString(line[i:], "")
}

// IncludeAt reports where the ssh_config text has `Include want`, want under ~/.ssh, and its line.
// The line counts by ~/.ssh/ or the home's full path too, in quotes, before a comment, or with
// other paths after it.
func IncludeAt(text, want, home string) (Place, int) {
	settings := false
	for n, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(strings.TrimLeft(line, " \t"), " \t\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, path := keyAndRest(line)
		if strings.HasPrefix(path, `"`) {
			path, _, _ = strings.Cut(path[1:], `"`)
		} else if i := strings.IndexAny(path, " \t"); i >= 0 {
			path = path[:i]
		}
		path = strings.TrimPrefix(strings.TrimPrefix(path, "~/.ssh/"), home+"/.ssh/")
		if strings.EqualFold(key, "include") && path == want {
			if settings {
				return Below, n + 1
			}
			return First, n + 1
		}
		settings = true
	}
	return Missing, 0
}
