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
		args := fields[1:]
		if strings.EqualFold(fields[0], "match") {
			args = matchedNames(args)
		}
		var hits []string
		for _, f := range args {
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

// matchedNames is the patterns that a Match line's host and originalhost criteria list. Another
// criterion's keyword or argument is no name, whatever the server is called.
func matchedNames(fields []string) []string {
	var out []string
	for i := 0; i < len(fields); i++ {
		switch key := strings.ToLower(fields[i]); key {
		case "all", "canonical", "final":
		case "host", "originalhost":
			if i++; i < len(fields) {
				out = append(out, strings.Split(fields[i], ",")...)
			}
		default:
			i++
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, e := range list {
		if e == s {
			return true
		}
	}
	return false
}
