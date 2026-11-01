package check

import "strings"

// KeyFile reports whether s is a public key's file, as ~/.ssh/NAME.pub: a path from / or ~/ to
// .pub, of letters, digits and ._/@+-.
func KeyFile(s string) bool {
	p := strings.TrimPrefix(s, "~")
	return strings.HasPrefix(p, "/") && strings.HasSuffix(p, ".pub") &&
		only(p, letters+digits+"._/@+-")
}

// Socket reports whether s is an agent's socket: a path from / or ~/, of letters, digits, blanks
// and ._/@+-. An app's socket may hold a blank, which needs quotes, as in a shell's assignment.
func Socket(s string, quoted bool) bool {
	p := strings.TrimPrefix(s, "~")
	if len(p) < 2 || p[0] != '/' || !only(p, letters+digits+"._/@+ -") {
		return false
	}
	return quoted || !strings.Contains(p, " ")
}

// Hook reports whether s is OWNER/REPO on GitHub, as ws's /root/repository/REPO: neither side . or
// .. alone, and REPO not ws, ws's own clone there, in any letter case, which APFS and GitHub ignore.
func Hook(s string) bool {
	owner, repo, found := strings.Cut(s, "/")
	return found && only(s, letters+digits+"._/-") && owner != "" && repo != "" &&
		!strings.Contains(repo, "/") && owner != "." && owner != ".." &&
		repo != "." && repo != ".." && !strings.EqualFold(repo, "ws")
}
