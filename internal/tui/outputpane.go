package tui

import (
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// clean leaves what a pane can draw of a line of a run. A step's output and a file's diff hold what
// a screen must not pass on: a code would move the cursor, and a control character would shift the
// frame. Of a line that \r overwrote, the last state stays, as a terminal would show it.
func clean(line string) string {
	line = strings.TrimRight(line, "\r")
	line = line[strings.LastIndexByte(line, '\r')+1:]
	line = strings.ReplaceAll(line, "\t", "    ")
	line = ansi.Strip(strings.ToValidUTF8(line, "�"))
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, line)
}

// style is how l draws. A diff's line goes by its first characters, which show under NO_COLOR too.
func (l Line) style(theme Theme) lipgloss.Style {
	switch l.Kind {
	case Warned:
		return theme.Warn
	case Piped:
		return theme.Muted
	case Diffed:
		switch {
		case strings.HasPrefix(l.Text, "+++"), strings.HasPrefix(l.Text, "---"), strings.HasPrefix(l.Text, "@@"):
			return theme.Muted
		case strings.HasPrefix(l.Text, "+"):
			return theme.Good
		case strings.HasPrefix(l.Text, "-"):
			return theme.Bad
		}
	case Said, Copied:
	}
	return lipgloss.NewStyle()
}
