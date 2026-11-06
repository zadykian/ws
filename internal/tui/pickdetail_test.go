package tui

import (
	"slices"
	"strings"
	"testing"
)

// detailOf is the words of the pane for the line that l's cursor stands on.
func detailOf(l pickList) []string {
	var got []string
	for _, line := range (pickScreen{list: l}).notes() {
		got = append(got, line.Text)
	}
	return got
}

func TestDetailShowsWhatTheScanSaid(t *testing.T) {
	l := chain().hear(Line{Diffed, "base: one: writes a\n+b"}).take(planned("base", "one", 0, oneChange("a")))
	want := []string{"writes a\n+b"}
	if got := detailOf(cursorOn(l, "one")); !slices.Equal(got, want) {
		t.Errorf("detail = %q, want %q", got, want)
	}
}

func TestDetailStaysInsideItsPane(t *testing.T) {
	said := Line{Said, strings.Repeat("a long line of a plan ", 40)}
	s := pickSent(testPick(), said, planned("base", "Packages", 0, oneChange("a")), press('j'))
	if got := len(strings.Split(pickShown(s), "\n")); got > 24 {
		t.Errorf("%d lines in a window of 24", got)
	}
}
