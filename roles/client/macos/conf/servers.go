package conf

// NameForm says what a server's name looks like, in the words of each refusal of a bad one.
const NameForm = "a lower-case letter, then up to 11 lower-case letters, digits and -"

// maxName is the longest a server's name goes, which the ssh alias and the daemon's suffix share.
const maxName = 12

// IsName reports whether s is a server's name.
func IsName(s string) bool {
	if s == "" || len(s) > maxName || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}
