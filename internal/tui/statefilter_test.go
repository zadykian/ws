package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/step"
)

// filtered is the screen with text typed into its filter.
func filtered(t *testing.T, text string) StateScreen {
	s, _ := feedState(t, reportedState(t), press('/'))
	for _, r := range text {
		s, _ = feedState(t, s, press(r))
	}
	return s
}

func TestStateFilterTakesEveryKeyItCan(t *testing.T) {
	s := filtered(t, "qr/")
	if got := s.filter.Value(); got != "qr/" {
		t.Errorf("the words %q", got)
	}
	if got, _ := feedState(t, reportedState(t), enter, press('/')); !got.editing || got.inDetail {
		t.Errorf("from the detail pane: editing %v, in the pane %v", got.editing, got.inDetail)
	}
	if _, cmd := feedState(t, s, press('q')); cmd != nil {
		t.Error("q left the screen from the filter")
	}
}

func TestStateFilterTakesAPasteWhileItHasTheKeys(t *testing.T) {
	paste := tea.PasteMsg{Content: "man"}
	if s, _ := feedState(t, reportedState(t), paste); s.filter.Value() != "" {
		t.Errorf("outside the filter, the words %q", s.filter.Value())
	}
	if s, _ := feedState(t, reportedState(t), press('/'), paste); s.filter.Value() != "man" {
		t.Errorf("the words %q", s.filter.Value())
	}
}

func TestStateScreenEscLeavesTheDetailThenDropsTheWords(t *testing.T) {
	s, _ := feedState(t, filtered(t, "pod"), enter, enter, esc)
	if s.inDetail || s.filter.Value() != "pod" {
		t.Fatalf("the first Esc: detail %v, the words %q", s.inDetail, s.filter.Value())
	}
	if s, _ = feedState(t, s, esc); s.filter.Value() != "" {
		t.Errorf("the second Esc leaves the words %q", s.filter.Value())
	}
}

func TestStateFilterKeepsTheSelectedRow(t *testing.T) {
	s, _ := feedState(t, reportedState(t), down, press('/'), press('p'))
	if got := selected(s); got != "podman" {
		t.Fatalf("under the filter, on %q", got)
	}
	s, _ = feedState(t, s, esc)
	if got := selected(s); got != "podman" {
		t.Errorf("without it, on %q", got)
	}
}

// wordsState is the screen on a list of packages by these names, with the last selected.
func wordsState(t *testing.T, rows ...string) StateScreen {
	items := make([]step.Item, len(rows))
	for i, name := range rows {
		items[i] = step.Item{Kind: step.PackageItem, Name: name}
	}
	read := StateRead{Sections: []StateSection{{Reports: []step.Report{{Step: "Packages", Items: items}}}}}
	s, _ := feedState(t, newTestState(), sized, read, press('G'))
	return s
}

// An edit before the end of the words rewrites the input's array, so that a screen from before it
// reads the new words. These two ran past the rows of those words.
func TestStateFilterTakesAnEditInsideItsWords(t *testing.T) {
	left, back := tea.KeyPressMsg{Code: tea.KeyLeft}, tea.KeyPressMsg{Code: tea.KeyBackspace}
	s, _ := feedState(t, wordsState(t, "a-ot", "b-oc", "c-oc", "d-oc", "e-oc", "f-oc", "g-oc"),
		press('/'), press('o'), press('c'), left, press('t'))
	if got := s.filter.Value(); got != "otc" || len(s.shown(packagesTab)) != 0 {
		t.Errorf("an insert: the words %q, rows %v", got, names(s.shown(packagesTab)))
	}
	s, _ = feedState(t, wordsState(t, "a-libb", "b-lxib", "c-lxib", "d-lxib"),
		press('/'), press('l'), press('x'), press('i'), press('b'), left, left, back)
	if got := selected(s); s.filter.Value() != "lib" || got != "a-libb" {
		t.Errorf("a delete: the words %q, on %q", s.filter.Value(), got)
	}
}

func TestStateFilterEditInsideItsWordsKeepsTheSelectedRow(t *testing.T) {
	s, _ := feedState(t, wordsState(t, "x-abc", "y-ac", "z-ac"), press('/'), press('a'), press('c'),
		tea.KeyPressMsg{Code: tea.KeyLeft}, tea.KeyPressMsg{Code: tea.KeyBackspace})
	if got := selected(s); s.filter.Value() != "c" || got != "z-ac" {
		t.Errorf("the words %q, on %q", s.filter.Value(), got)
	}
}
