package base

import "strings"

// supportedFile lists the locales glibc can generate, as the locales package ships it.
const supportedFile = "/usr/share/i18n/SUPPORTED"

// entry returns the line of /etc/locale.gen for the name, which is the line of supportedFile
// that starts with it: the name and its code set. It reports false where the file has no such
// locale.
func entry(supported, name string) (string, bool) {
	for _, line := range strings.Split(supported, "\n") {
		f := strings.Fields(line)
		if len(f) == 2 && f[0] == name {
			return f[0] + " " + f[1], true
		}
	}
	return "", false
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
