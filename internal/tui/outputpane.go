package tui

import (
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	// outputCap is the lines that a pane keeps, as a first run prints more than memory should hold.
	outputCap = 10_000
	// outputSlack is how far past outputCap the lines grow before the oldest go, as that draws the
	// rest again.
	outputSlack = 2_000
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

// outputPane is a window on what a run prints. It follows the end until a key moves it, and then
// holds still as lines arrive. rows are the lines as drawn, wrapped to the pane, as a pane cuts what
// is wider, and each is styled alone, so that no color crosses a line's end.
type outputPane struct {
	theme Theme
	lines []Line
	rows  []string
	size  Size
	// back is the rows between the last one in sight and the end.
	back int
}

// add puts l's lines at the end.
func (o outputPane) add(l Line) outputPane {
	for text := range strings.SplitSeq(l.Text, "\n") {
		line := Line{l.Kind, clean(text)}
		drawn := o.draw(line)
		o.lines, o.rows = append(o.lines, line), append(o.rows, drawn...)
		if o.back > 0 {
			o.back += len(drawn)
		}
	}
	if len(o.lines) > outputCap+outputSlack {
		o.lines = o.lines[len(o.lines)-outputCap:]
		return o.fit(o.size)
	}
	return o
}

// draw returns l's rows at the pane's width.
func (o outputPane) draw(l Line) []string {
	style := l.style(o.theme)
	rows := strings.Split(lipgloss.Wrap(l.Text, o.size.Width, ""), "\n")
	for i, row := range rows {
		rows[i] = style.Render(row)
	}
	return rows
}

// fit draws the lines for a pane of size.
func (o outputPane) fit(size Size) outputPane {
	o.size = size
	o.rows = make([]string, 0, len(o.lines))
	for _, l := range o.lines {
		o.rows = append(o.rows, o.draw(l)...)
	}
	return o.move(0)
}

// move goes back by rows, or towards the end where they are negative, as far as there is to see.
func (o outputPane) move(rows int) outputPane {
	o.back = min(max(o.back+rows, 0), max(len(o.rows)-o.size.Height, 0))
	return o
}

// view draws the rows in sight.
func (o outputPane) view() string {
	end := len(o.rows) - o.back
	return strings.Join(o.rows[max(end-o.size.Height, 0):end], "\n")
}
