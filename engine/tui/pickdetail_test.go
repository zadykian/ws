package tui

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// detailOf is the words of the pane for the step of that name in l.
func detailOf(l pickList, name string) []string {
	var got []string
	for _, line := range (pickScreen{list: l, at: cursorOn(l, name)}).notes() {
		got = append(got, line.Text)
	}
	return got
}

func TestDetailSaysWhyAStepIsLocked(t *testing.T) {
	l := choosing(chain(), "three")
	want := []string{"Locked: three waits on it.", "Waits on one.", "Not scanned yet."}
	if got := detailOf(l, "two"); !slices.Equal(got, want) {
		t.Errorf("detail = %q, want %q", got, want)
	}
}

func TestDetailShowsWhatTheScanSaid(t *testing.T) {
	l := chain().hear(widget.Line{Kind: widget.Diffed, Text: "base: one: writes a\n+b"}).take(planned("base", "one", 0, oneChange("a")))
	want := []string{"Holds changes.", "writes a\n+b"}
	if got := detailOf(l, "one"); !slices.Equal(got, want) {
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
		if got := detailOf(chain().take(c.e), "one"); len(got) != 1 || got[0] != c.want {
			t.Errorf("detail = %q, want %q", got, c.want)
		}
	}
}

func TestDetailWarnsOfARolePassedOverAndOfAStoppedScan(t *testing.T) {
	l := threeRoles().chose(func(r pickRow) bool { return r.name == "four" })
	s := testPick()
	s.list, s.scanning, s.problem = l, false, "no network"
	got := pickShown(pickSent(s, tea.WindowSizeMsg{Width: 100, Height: 24}))
	for _, want := range []string{"! 1 of 4 chosen", "The scan stopped: no network", "changes left out in a, b"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in:\n%s", want, got)
		}
	}
}

func TestDetailStaysInsideItsPane(t *testing.T) {
	said := widget.Line{Kind: widget.Said, Text: strings.Repeat("a long line of a plan ", 40)}
	s := pickSent(testPick(), said, planned("base", "Packages", 0, oneChange("a")), press('j'))
	if got := len(strings.Split(pickShown(s), "\n")); got > 24 {
		t.Errorf("%d lines in a window of 24", got)
	}
}
