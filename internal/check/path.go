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

// Hook reports whether s is OWNER/REPO@TAG, a release of a repository on GitHub, whose command
// REPO ws runs: neither side of the / is . or .. alone, and REPO is not ws, in any letter case,
// which APFS and GitHub ignore. The tag is required.
func Hook(s string) bool {
	s, tag, tagged := strings.Cut(s, "@")
	if !tagged || !Release(tag) {
		return false
	}
	owner, repo, found := strings.Cut(s, "/")
	return found && only(s, letters+digits+"._/-") && owner != "" && repo != "" &&
		!strings.Contains(repo, "/") && owner != "." && owner != ".." &&
		repo != "." && repo != ".." && !strings.EqualFold(repo, "ws")
}
