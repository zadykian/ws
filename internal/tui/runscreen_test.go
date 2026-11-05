package tui

import (
	"testing"
	"time"
)

func TestBar(t *testing.T) {
	cases := []struct {
		done, total, width int
		want               string
	}{
		{0, 4, 4, "░░░░"},
		{1, 4, 4, "█░░░"},
		{4, 4, 4, "████"},
		{0, 0, 4, "░░░░"},
	}
	for _, c := range cases {
		if got := bar(c.done, c.total, c.width); got != c.want {
			t.Errorf("bar(%d, %d, %d) = %q, want %q", c.done, c.total, c.width, got, c.want)
		}
	}
}

func TestElapsed(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "00:00"},
		{59 * time.Second, "00:59"},
		{20*time.Minute + 5*time.Second, "20:05"},
		{time.Hour + 2*time.Minute + 3*time.Second, "1:02:03"},
	}
	for _, c := range cases {
		if got := elapsed(c.d); got != c.want {
			t.Errorf("elapsed(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}
