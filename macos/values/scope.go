package values

import (
	"slices"
	"strings"
)

// Scope is the file that a text is: the Mac's values.yml, or a server's file beside it. Each file
// refuses the other's keys.
type Scope int

const (
	// Both takes any key: the zero value, for a text not yet known to be one file's.
	Both Scope = iota
	// MacFile takes the Mac's keys alone.
	MacFile
	// ServerFile takes every key but the Mac's.
	ServerFile
)

// macKeys are the keys of values.yml. Every other key is a server's, so a new key is one unless it
// is listed here. The Mac's file takes none now.
var macKeys []string

// refuse returns the message for the key name where the scope is the other file's, else nothing.
func (s Scope) refuse(name string) string {
	mac := slices.Contains(macKeys, name)
	switch {
	case s == MacFile && !mac:
		return name + " is a server's key: it goes in servers/NAME/values.yml, beside the Mac's values.yml"
	case s == ServerFile && mac:
		return name + " is the Mac's key: it goes in the Mac's values.yml"
	}
	return ""
}

// KeyScope returns the file that the key goes in: MacFile for the Mac's keys, else ServerFile.
func KeyScope(key string) Scope {
	if slices.Contains(macKeys, key) {
		return MacFile
	}
	return ServerFile
}

// Keep returns the KEY: value lines of the text whose key goes in the scope, in order. Both keeps
// every line.
func (s Scope) Keep(lines string) string {
	var kept strings.Builder
	for _, line := range strings.SplitAfter(lines, "\n") {
		key, _ := lineKey(line)
		if s == Both || KeyScope(key) == s {
			kept.WriteString(line)
		}
	}
	return kept.String()
}
