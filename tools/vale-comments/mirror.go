package main

import (
	"strings"
	"unicode/utf8"
)

// An origin is where a mirror's line comes from: the source's line, and the rune column at which
// the mirrored text starts in it. The zero origin is a blank line's, which comes from no comment.
type origin struct{ line, col int }

// A mirror is a file's comments as Markdown, a line for each comment line, with its origin. A
// blank line ends a comment, so that Vale reads two comments as two paragraphs. texts counts the
// lines that hold text.
type mirror struct {
	lines []string
	from  []origin
	texts int
}

// newMirror starts with a blank line, lest a first comment of dashes start front matter.
func newMirror() *mirror {
	return &mirror{lines: []string{""}, from: []origin{{}}}
}

func (m *mirror) add(text string, o origin) {
	m.lines = append(m.lines, text)
	m.from = append(m.from, o)
	if strings.TrimSpace(text) != "" {
		m.texts++
	}
}

// gap ends the current paragraph.
func (m *mirror) gap() {
	if m.lines[len(m.lines)-1] != "" {
		m.add("", origin{})
	}
}

// text returns what a comment's body, after its marker, gives Vale, and the runes before it.
func text(body string) (string, int) {
	rest := strings.TrimPrefix(body, " ")
	return rest, len(body) - len(rest)
}

// lineMirror mirrors the comments that marker starts, of full lines alone: a line rule cannot tell
// a trailing comment from a marker in a string.
func lineMirror(src []byte, marker string) *mirror {
	m := newMirror()
	for i, line := range strings.Split(string(src), "\n") {
		rest := strings.TrimLeft(line, " \t")
		body, found := strings.CutPrefix(rest, marker)
		body, skip := text(body)
		if !found || i == 0 && strings.HasPrefix(line, "#!") {
			m.gap()
			continue
		}
		indent := utf8.RuneCountInString(line[:len(line)-len(rest)])
		m.add(body, origin{i + 1, indent + len(marker) + skip + 1})
	}
	return m
}
