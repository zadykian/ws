package gitcfg

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	keyType   = regexp.MustCompile(`^(ssh-|ecdsa-sha2-|sk-).`)
	keyBase64 = regexp.MustCompile(`^[A-Za-z0-9+/=]+$`)
)

// PublicKey returns the type and base64 of the public key that a .pub holds, or empty where the
// file holds more or less than the line of one key, as a private key does, or a line that is no key.
func PublicKey(pub string) string {
	var key string
	lines := 0
	for _, line := range strings.Split(pub, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		lines++
		if len(fields) > 1 {
			key = fields[0] + " " + fields[1]
		} else {
			key = fields[0] + " "
		}
	}
	typ, data, _ := strings.Cut(key, " ")
	if lines != 1 || !keyType.MatchString(typ) || !keyBase64.MatchString(data) {
		return ""
	}
	return key
}

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
