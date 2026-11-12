package gitcfg

import (
	"strconv"
	"strings"
)

// Why says why gh cannot register the key, from gh's exit status and its errors, in the devtools
// role's words. A 404 answers a login without the scope, and gh's last line of error loses its
// final period.
func Why(status int, stderr string) string {
	switch {
	case status == 4:
		return "gh has no login: gh auth login -h github.com -s write:ssh_signing_key logs in"
	case strings.Contains(stderr, "HTTP 404"):
		return "gh's login lacks the write:ssh_signing_key scope: gh auth refresh -h github.com " +
			"-s write:ssh_signing_key adds it"
	}
	lines := strings.Split(strings.TrimRight(stderr, "\n"), "\n")
	last := strings.TrimSuffix(lines[len(lines)-1], ".")
	if last == "" {
		return "gh failed: exit " + strconv.Itoa(status)
	}
	return "gh failed: " + last
}
