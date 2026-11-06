package tui

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/step"
)

// detailOf is the words of the pane for the line that l's cursor stands on.
func detailOf(l pickList) []string {
	var got []string
	for _, line := range (pickScreen{list: l}).notes() {
		got = append(got, line.Text)
	}
	return got
}

func TestDetailSaysWhyAStepIsLocked(t *testing.T) {
	l := cursorOn(choosing(chain(), "three"), "two")
	want := []string{"Locked: three waits on it.", "Waits on one.", "Not scanned yet."}
	if got := detailOf(l); !slices.Equal(got, want) {
		t.Errorf("detail = %q, want %q", got, want)
	}
}

func TestDetailShowsWhatTheScanSaid(t *testing.T) {
	l := chain().hear(Line{Diffed, "base: one: writes a\n+b"}).take(planned("base", "one", 0, oneChange("a")))
	want := []string{"Holds changes.", "writes a\n+b"}
	if got := detailOf(cursorOn(l, "one")); !slices.Equal(got, want) {
		t.Errorf("detail = %q, want %q", got, want)
	}
}

func TestDetailWordsEachStateOfTheScan(t *testing.T) {
	cases := []struct {
		e    step.Event
		want string
	}{
		{planned("base", "one", 0), "Nothing to change."},
		{step.Event{Kind: step.Failed, Parent: "base", Step: "one"}, "The scan failed here."},
		{step.Event{Kind: step.Skipped, Parent: "base", Step: "one"}, "Not scanned: the scan ended first."},
		{step.Event{Kind: step.Skipped, Parent: "base", Step: "one", After: "zero"}, "Not planned: it follows the changes of zero."},
	}
	for _, c := range cases {
		if got := detailOf(cursorOn(chain().take(c.e), "one")); len(got) != 1 || got[0] != c.want {
			t.Errorf("detail = %q, want %q", got, c.want)
		}
	}
}

func TestDetailStaysInsideItsPane(t *testing.T) {
	said := Line{Said, strings.Repeat("a long line of a plan ", 40)}
	s := pickSent(testPick(), said, planned("base", "Packages", 0, oneChange("a")), press('j'))
	if got := len(strings.Split(pickShown(s), "\n")); got > 24 {
		t.Errorf("%d lines in a window of 24", got)
	}
}
