package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// wantDrawn checks that the screen, as a person reads it, holds the words.
func wantDrawn(t *testing.T, s StateScreen, words ...string) {
	got := stripped(s.View().Content)
	for _, word := range words {
		if !strings.Contains(got, word) {
			t.Errorf("no %q in:\n%s", word, got)
		}
	}
}

func TestStateScreenDrawsTheTabs(t *testing.T) {
	s, _ := feedState(t, newTestState(), sized)
	wantDrawn(t, s, " Packages  ", "reading")
	wantDrawn(t, reportedState(t), " Packages 2 ", " Services 1 ", " Files 3 ", " Conflicts 2 ", "ws server state")
	s, _ = feedState(t, reportedState(t), tea.WindowSizeMsg{Width: 30, Height: 24})
	wantDrawn(t, s, " Packages 2 ", " Services 1 ", " Files 3 ", " Conflicts 2 ")
}

func TestStateScreenSaysWhyAReadFailed(t *testing.T) {
	s, _ := feedState(t, reportedState(t), Line{Warned, "a slow disk"}, StateRead{Err: errors.New("no ssh")})
	wantDrawn(t, s, "no ssh", "a slow disk", " Packages 2 ")
}

func TestStateScreenDrawsInAnyWindow(t *testing.T) {
	s := reportedState(t)
	for w := range 90 {
		for h := range 30 {
			s, _ = feedState(t, s, tea.WindowSizeMsg{Width: w, Height: h})
			got := s.View().Content
			if width, height := lipgloss.Width(got), lipgloss.Height(got); width > w || height > max(h, 1) {
				t.Errorf("%dx%d: drew %dx%d", w, h, width, height)
			}
		}
	}
}
