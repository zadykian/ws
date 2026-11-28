package check

import "strings"

// Domains reports whether list is DNS names split by blanks, each once and in lower case. Two
// spellings would share one file of /etc/resolver, whose volume ignores case by default.
func Domains(list string) bool {
	if strings.Trim(list, "abcdefghijklmnopqrstuvwxyz0123456789.- \t") != "" {
		return false
	}
	seen := map[string]bool{}
	for _, domain := range strings.Fields(list) {
		if !Name(domain) || seen[domain] {
			return false
		}
		seen[domain] = true
	}
	return true
}
