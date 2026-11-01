package sshcfg

import "strings"

// OwnHost is a line that starts a Host or Match block for one of ws's names.
type OwnHost struct {
	Line  int
	Names string // those it matches, joined by " and "
	Text  string
}

// OwnHosts finds each line of the ssh_config text that starts a Host or Match block for one of the
// names. Its settings come after those of the Include at the file's top.
func OwnHosts(text string, names ...string) []OwnHost {
	var found []OwnHost
	for n, line := range strings.Split(text, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || (!strings.EqualFold(fields[0], "host") && !strings.EqualFold(fields[0], "match")) {
			continue
		}
		var hits []string
		for _, f := range fields[1:] {
			for _, name := range names {
				if f == name && !contains(hits, f) {
					hits = append(hits, f)
				}
			}
		}
		if len(hits) > 0 {
			found = append(found, OwnHost{n + 1, strings.Join(hits, " and "), line})
		}
	}
	return found
}

func contains(list []string, s string) bool {
	for _, e := range list {
		if e == s {
			return true
		}
	}
	return false
}
