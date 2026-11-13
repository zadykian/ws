package tuitest

import "regexp"

// LeaveBuffer is what puts a terminal back on its normal screen.
const LeaveBuffer = "\x1b[?1049l"

// SGR finds the parameters of a text style's escape sequence.
var SGR = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

// Stripped drops the text styles from a drawing, to compare its words.
func Stripped(text string) string {
	return SGR.ReplaceAllString(text, "")
}

// NoColor is the getenv of an environment with NO_COLOR=value.
func NoColor(value string) func(string) string {
	return func(string) string { return value }
}
