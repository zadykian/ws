package base

import (
	"regexp"
	"strings"
)

// localeName is a glibc locale: language_TERRITORY.CODESET and an optional @modifier.
var localeName = regexp.MustCompile(`^[a-z]{2,3}_[A-Z]{2}\.[A-Za-z0-9-]+(@[a-z]+)?$`)

// entry returns the line of /etc/locale.gen: the name and its code set, UTF-8 for en_US.UTF-8.
func entry(name string) string {
	_, rest, _ := strings.Cut(name, ".")
	charset, _, _ := strings.Cut(rest, "@")
	return name + " " + charset
}

// listed reports whether `locale -a`'s output has the locale under the name glibc gives it: the
// code set in lower case without punctuation, so that en_US.UTF-8 is en_US.utf8.
func listed(out, name string) bool {
	lang, rest, hasSet := strings.Cut(name, ".")
	if hasSet {
		set, mod, hasMod := strings.Cut(rest, "@")
		set = strings.ToLower(strings.Map(func(r rune) rune {
			if r == '-' || r == '_' {
				return -1
			}
			return r
		}, set))
		if set != "" && set[0] >= '0' && set[0] <= '9' {
			set = "iso" + set
		}
		name = lang + "." + set
		if hasMod {
			name += "@" + mod
		}
	}
	for _, l := range strings.Fields(out) {
		if l == name {
			return true
		}
	}
	return false
}
