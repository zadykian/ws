package tui

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func testPane(size Size, texts ...string) outputPane {
	o := outputPane{theme: testTheme}.fit(size)
	for _, text := range texts {
		o = o.add(Line{Said, text})
	}
	return o
}

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

func TestOutputPaneFollowsTheEnd(t *testing.T) {
	o := testPane(Size{10, 2}, "one", "two", "three")
	if got, want := o.view(), "two\nthree"; got != want {
		t.Errorf("view = %q, want %q", got, want)
	}
}

func TestOutputPaneSplitsAndWrapsALine(t *testing.T) {
	o := testPane(Size{5, 4}, "one\ntwo words")
	if got, want := o.view(), "one\ntwo\nwords"; got != want {
		t.Errorf("view = %q, want %q", got, want)
	}
}

func TestOutputPaneHoldsStillOnceMoved(t *testing.T) {
	o := testPane(Size{10, 2}, "one", "two", "three", "four").move(1)
	want := "two\nthree"
	if got := o.view(); got != want {
		t.Fatalf("view = %q, want %q", got, want)
	}
	if got := o.add(Line{Said, "five"}).view(); got != want {
		t.Errorf("view past a new line = %q, want %q", got, want)
	}
	if got := o.add(Line{Said, "five"}).move(-5).view(); got != "four\nfive" {
		t.Errorf("view at the end = %q, want the end followed again", got)
	}
}

func TestOutputPaneStopsAtTheTop(t *testing.T) {
	o := testPane(Size{10, 2}, "one", "two", "three").move(100)
	if got, want := o.view(), "one\ntwo"; got != want {
		t.Errorf("view = %q, want %q", got, want)
	}
}

func TestOutputPaneShorterThanItsWindow(t *testing.T) {
	o := testPane(Size{10, 5}, "one").move(3)
	if got := o.view(); got != "one" {
		t.Errorf("view = %q, want the one line", got)
	}
}

func TestOutputPaneFitWrapsAgain(t *testing.T) {
	o := testPane(Size{20, 3}, "two words").fit(Size{5, 3})
	if got, want := o.view(), "two\nwords"; got != want {
		t.Errorf("view = %q, want %q", got, want)
	}
}

func TestOutputPaneKeepsTheEndOfALongRun(t *testing.T) {
	o := testPane(Size{20, 1})
	for i := range outputCap + outputSlack + 1 {
		o = o.add(Line{Said, fmt.Sprint("line ", i)})
	}
	if len(o.lines) != outputCap || len(o.rows) != outputCap {
		t.Errorf("lines, rows = %d, %d, want %d", len(o.lines), len(o.rows), outputCap)
	}
	if got, want := o.view(), fmt.Sprint("line ", outputCap+outputSlack); got != want {
		t.Errorf("view = %q, want %q", got, want)
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

func TestOutputPaneStylesEachRowAlone(t *testing.T) {
	theme := NewTheme(func(string) string { return "" })
	o := outputPane{theme: theme}.fit(Size{5, 3}).add(Line{Warned, "two words"})
	for _, row := range strings.Split(o.view(), "\n") {
		if !strings.HasSuffix(row, "m") {
			t.Errorf("row %q holds no end of its color", row)
		}
	}
}
