package tui

import (
	"testing"

	"charm.land/lipgloss/v2"
)

func TestClean(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"plain", "a line", "a line"},
		{"color", "\x1b[31mred\x1b[0m", "red"},
		{"cursor", "\x1b[2J\x1b[Hhome", "home"},
		{"title", "\x1b]0;title\x07text", "text"},
		{"bell and nul", "a\x07b\x00c\x7fd", "abcd"},
		{"line end of a pipe", "line\r", "line"},
		{"a meter", "10%\r50%\r100%", "100%"},
		{"a tab", "a\tb", "a    b"},
		{"a C1 code", "a\u009b31mb", "a31mb"},
		{"a bad byte", "a\xffb", "a�b"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := clean(c.in); got != c.want {
				t.Errorf("clean(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestLineStyleColorsADiff(t *testing.T) {
	theme := NewTheme(func(string) string { return "" })
	cases := []struct {
		text string
		want lipgloss.Style
	}{
		{"+added", theme.Good},
		{"-removed", theme.Bad},
		{"@@ -1 +1 @@", theme.Muted},
		{"--- a", theme.Muted},
		{"+++ a", theme.Muted},
		{" context", lipgloss.NewStyle()},
	}
	for _, c := range cases {
		if got := (Line{Diffed, c.text}).style(theme).Render("x"); got != c.want.Render("x") {
			t.Errorf("%q draws as %q, want %q", c.text, got, c.want.Render("x"))
		}
	}
}

func TestLineStyleLeavesAPlainLineAlone(t *testing.T) {
	theme := NewTheme(func(string) string { return "" })
	if got := (Line{Said, "+1 change"}).style(theme).Render("x"); got != "x" {
		t.Errorf("a said line draws as %q, want no color", got)
	}
}
