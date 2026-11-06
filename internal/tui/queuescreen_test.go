package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// testQueueScreen is the two lists at 100 by 30, their scan under way.
func testQueueScreen() pickScreen {
	s := newQueueScreen("ws mac wizard", testTheme, testStages(), make(chan Choice, 1))
	return pickSent(s, tea.WindowSizeMsg{Width: 100, Height: 30})
}

func TestTheTwoListsStandSideBySideOverTheDetail(t *testing.T) {
	got := pickShown(testQueueScreen())
	for _, want := range []string{"ws mac wizard", "scanning 0/5", "Mac", "ws: roles not scanned", "Detail",
		"▸  1 [ ] · Tools", "   3 [ ] · Sudo", "Not scanned yet."} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in:\n%s", want, got)
		}
	}
	lines := strings.Split(got, "\n")
	if i := strings.Index(lines[2], "Tools"); i < 0 || !strings.Contains(lines[2][i:], "Sudo") {
		t.Errorf("not side by side:\n%s", got)
	}
}

func TestAShortWindowHasNoStrip(t *testing.T) {
	got := pickShown(pickSent(testQueueScreen(), tea.WindowSizeMsg{Width: 100, Height: 12}))
	if strings.Contains(got, "Detail") || !strings.Contains(got, "Sudo") {
		t.Errorf("frame:\n%s", got)
	}
}

func TestANarrowWindowStacksTheLists(t *testing.T) {
	got := pickShown(pickSent(testQueueScreen(), tea.WindowSizeMsg{Width: 40, Height: 30}))
	if strings.Index(got, "Tools") > strings.Index(got, "Sudo") || !strings.Contains(got, "Detail") {
		t.Errorf("frame:\n%s", got)
	}
}

func TestAStripTakesAThirdOfTheWindowWithinBounds(t *testing.T) {
	cases := []struct {
		height, want int
	}{{13, 0}, {14, stripLeast}, {24, 8}, {60, stripMost}}
	for _, c := range cases {
		if got := stripSize(Size{80, c.height}).Height; got != c.want {
			t.Errorf("a window of %d lines has a strip of %d, want %d", c.height, got, c.want)
		}
	}
}

func TestAReadRoleClearsTheTitleOfWsList(t *testing.T) {
	stages := []Stage{{Name: "Run", Side: WsSide, Nested: []Group{{Name: "base"}}}}
	l := newQueue(stages)
	if got := l.panes()[WsSide].Title; got != "ws: roles not scanned" {
		t.Errorf("title = %q", got)
	}
	if got := l.reported("Run", nil).panes()[WsSide].Title; got != "ws" {
		t.Errorf("title = %q", got)
	}
}
