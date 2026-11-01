package sshcfg

import (
	"regexp"
	"strings"
)

var (
	trailingCmt = regexp.MustCompile(`[ \t]#.*$`)
	patternSep  = regexp.MustCompile(`[, \t]+`)
)

// IgnoreUnknown returns first and the patterns of every IgnoreUnknown of the ssh_config text, in any
// block, each once, joined by commas. ssh keeps the first IgnoreUnknown it reads, so the one in a
// file included above the text's settings must carry them all.
func IgnoreUnknown(text, first string) string {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		if p != "" && !seen[strings.ToLower(p)] {
			seen[strings.ToLower(p)] = true
			out = append(out, p)
		}
	}
	add(first)
	for _, line := range strings.Split(text, "\n") {
		key, patterns := keyAndRest(strings.TrimLeft(line, " \t"))
		if !strings.EqualFold(key, "ignoreunknown") {
			continue
		}
		patterns = strings.TrimRight(trailingCmt.ReplaceAllString(patterns, ""), " \t\r")
		for _, p := range patternSep.Split(strings.ReplaceAll(patterns, `"`, ""), -1) {
			add(p)
		}
	}
	return strings.Join(out, ",")
}
