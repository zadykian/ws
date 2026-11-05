package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

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
	if s, _ = feedState(t, newTestState(), sized, StateRead{}, enter); s.inDetail {
		t.Error("an empty tab opens the detail pane")
	}
}
