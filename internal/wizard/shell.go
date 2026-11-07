package wizard

import "strings"

// shellWord returns word as sh reads it back: bare where it holds only letters, digits and
// -_./=:,+@, else in single quotes, with each quote of its own closed, escaped and opened again.
func shellWord(word string) string {
	if word != "" && strings.Trim(word, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_./=:,+@") == "" {
		return word
	}
	return "'" + strings.ReplaceAll(word, "'", `'\''`) + "'"
}
