package state

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui/internal/run"
)

// fileReport is a report of the file /a, with this diff, and of the file /b.
func fileReport(diff string) Read {
	items := []step.Item{{Kind: step.FileItem, Name: "/a", Diff: diff}, {Kind: step.FileItem, Name: "/b"}}
	return Read{Sections: []run.StateSection{{Reports: []step.Report{{Items: items}}}}}
}

func TestStateScreenDrawsTheSelectedRow(t *testing.T) {
	s, _ := feedState(t, reportedState(t), down)
	wantDrawn(t, s, "Kind: package", "Step: docker/Packages", "Want: missing", "Found: 5.0", "Conflict: held by hand")
}

func TestStateScreenDetailTakesTheKeys(t *testing.T) {
	s, _ := feedState(t, reportedState(t), enter, down)
	if !s.layout().Panes[detailPane].Focused || selected(s) != "docker-ce" {
		t.Fatalf("in the detail pane %v, on %q", s.inDetail, selected(s))
	}
	for _, back := range []tea.Msg{esc, press('h'), tea.KeyPressMsg{Code: tea.KeyLeft}, tab} {
		if got, _ := feedState(t, s, back); got.inDetail {
			t.Errorf("%v stays in the detail pane", back)
		}
	}
	if _, cmd := feedState(t, s, press('q')); cmd == nil {
		t.Error("q stays in the detail pane")
	}
	if s, _ = feedState(t, newTestState(), sized, Read{}, enter); s.inDetail {
		t.Error("an empty tab opens the detail pane")
	}
}

func TestStateScreenColorsADiff(t *testing.T) {
	s, _ := feedState(t, reportedState(t), tab, tab)
	got := s.View().Content
	for _, want := range []string{
		testTheme.Muted.Render("@@ -1 +1 @@"), testTheme.Good.Render("+new"), testTheme.Bad.Bold(false).Render("-old"),
	} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in %q", want, got)
		}
	}
}

func TestDiffStyleTakesALaterMarkForAChange(t *testing.T) {
	cases := map[string]lipgloss.Style{
		"--- a removed line": testTheme.Bad.Bold(false),
		"+++ an added line":  testTheme.Good,
		" a line that stays": lipgloss.NewStyle(),
	}
	for line, want := range cases {
		if got := newTestState().diffStyle(2, line); got.Render("x") != want.Render("x") {
			t.Errorf("%q has another style", line)
		}
	}
}

func TestStateScreenScrollsTheDetail(t *testing.T) {
	var diff strings.Builder
	for i := range 60 {
		fmt.Fprintf(&diff, "+line %d \n", i)
	}
	s, _ := feedState(t, newTestState(), sized, fileReport(diff.String()), tab, tab)
	height := s.layout().PaneSizes(80, 24)[detailPane].Height
	wantDrawn(t, s, fmt.Sprintf("Detail 1-%d/", height), "+line 0 ")
	s, _ = feedState(t, s, enter, down)
	wantDrawn(t, s, fmt.Sprintf("Detail 2-%d/", height+1))
	s, _ = feedState(t, s, press('G'))
	wantDrawn(t, s, "+line 59 ")
	s, _ = feedState(t, s, esc, down, press('k'))
	if s.scroll != 0 {
		t.Errorf("a new row shows from line %d", s.scroll)
	}
}

func TestStateScreenWrapsADiffLine(t *testing.T) {
	s, _ := feedState(t, newTestState(), sized, fileReport("+"+strings.Repeat("word ", 40)), tab, tab)
	lines := s.detailLines(s.layout().PaneSizes(80, 24)[detailPane].Width)
	if got := strings.Count(strings.Join(lines, " "), "word"); got != 40 {
		t.Errorf("%d of the 40 words stay", got)
	}
}
