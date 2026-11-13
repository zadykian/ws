package tui

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zadykian/ws/engine/core/step"
)

// colored finds a text style that writes a color, whose parameter takes two digits.
var colored = regexp.MustCompile(`\x1b\[([0-9]+;)*[0-9]{2}(;[0-9]+)*m`)

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

func TestStateScreenDrawsTheRows(t *testing.T) {
	wantDrawn(t, reportedState(t), "docker-ce", "! podman")
	s, _ := feedState(t, newTestState(), sized)
	wantDrawn(t, s, readingNote)
	s, _ = feedState(t, s, StateRead{})
	wantDrawn(t, s, " Packages 0 ", emptyNote)
}

func TestStateScreenKeepsItsMarksWithoutColor(t *testing.T) {
	s, _ := feedState(t, NewStateScreen(NewTheme(noColor("1")), "ws", nil, nil), sized,
		StateRead{Sections: testSections()})
	if got := s.View().Content; colored.MatchString(got) {
		t.Errorf("a color under NO_COLOR: %q", got)
	}
	wantDrawn(t, s, "! podman", "Conflicts 2")
}

func TestStateScreenShowsTheSelectionInALongList(t *testing.T) {
	var items []step.Item
	for i := range 100 {
		items = append(items, step.Item{Kind: step.PackageItem, Name: fmt.Sprintf("row-%d.", i)})
	}
	report := StateRead{Sections: []StateSection{{Reports: []step.Report{{Items: items}}}}}
	s, _ := feedState(t, newTestState(), sized, report)
	wantDrawn(t, s, "row-0.")
	s, _ = feedState(t, s, press('G'))
	wantDrawn(t, s, "row-99.")
	s, _ = feedState(t, s, tea.KeyPressMsg{Code: tea.KeyPgUp})
	wantDrawn(t, s, selected(s))
	if got, want := s.cursors[packagesTab], 99-s.layout().PaneSizes(80, 24)[listPane].Height; got != want {
		t.Errorf("on row %d, want %d", got, want)
	}
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
