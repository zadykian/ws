package tui

import "testing"

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
