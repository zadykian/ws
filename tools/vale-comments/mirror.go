package main

import (
	"regexp"
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

var (
	// hashDirective matches the body of ShellCheck's, yamllint's and zizmor's directives, and not
	// a comment's sentence that starts with the linter's name.
	hashDirective = regexp.MustCompile(`^\s*(?:shellcheck\s+[a-z-]+=|yamllint\s+(?:disable|enable)|zizmor:)`)
)

// text returns the text that a comment's body, after its marker, gives Vale, and the runes of the
// body before it. A directive gives the reason after its own marker alone, and false without one.
func text(body string, directive *regexp.Regexp, marker string) (string, int, bool) {
	skip := 0
	if directive != nil && directive.MatchString(body) {
		i := strings.Index(body, " "+marker)
		if i < 0 {
			return "", 0, false
		}
		skip = utf8.RuneCountInString(body[:i]) + 1 + len(marker)
		body = body[i+1+len(marker):]
	}
	rest := strings.TrimPrefix(body, " ")
	return rest, skip + len(body) - len(rest), true
}

// lineMirror mirrors the comments that marker starts, of full lines alone: a line rule cannot tell
// a trailing comment from a marker in a string.
func lineMirror(src []byte, marker string, directive *regexp.Regexp) *mirror {
	m := newMirror()
	for i, line := range strings.Split(string(src), "\n") {
		rest := strings.TrimLeft(line, " \t")
		body, found := strings.CutPrefix(rest, marker)
		body, skip, ok := text(body, directive, marker)
		if !found || !ok || i == 0 && strings.HasPrefix(line, "#!") {
			m.gap()
			continue
		}
		indent := utf8.RuneCountInString(line[:len(line)-len(rest)])
		m.add(body, origin{i + 1, indent + len(marker) + skip + 1})
	}
	return m
}
