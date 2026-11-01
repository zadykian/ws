package check

import "strings"

// User reports whether s is a user name that ssh takes: letters, digits, ., _ and -, with no -
// first, and 32 characters at most.
func User(s string) bool {
	return s != "" && s[0] != '-' && len(s) <= 32 &&
		strings.Trim(s, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789._-") == ""
}
