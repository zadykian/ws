package tui

import (
	"strings"
	"unicode"
)

// firstLine returns the text up to its first line's end.
func firstLine(text string) string {
	line, _, _ := strings.Cut(text, "\n")
	return line
}

// printable makes text safe to draw. A pane measures columns, which a tab has none of, and what a
// file holds must not reach the terminal as a control code.
func printable(text string) string {
	text = strings.ReplaceAll(strings.ToValidUTF8(text, "�"), "\t", "    ")
	return strings.Map(func(r rune) rune {
		if r != '\n' && unicode.IsControl(r) {
			return -1
		}
		return r
	}, text)
}
