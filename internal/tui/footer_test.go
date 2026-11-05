package tui

import (
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

var (
	tab      = tea.KeyPressMsg{Code: tea.KeyTab}
	shiftTab = tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
)

// testFooter holds the buttons run, on Enter, check and quit, at places 0, 1 and 2.
func testFooter() Footer {
	return NewFooter(
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "run")),
		key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "check")),
		Keys.Quit,
	)
}

func TestFooterTabWalksThePanesAndTheButtons(t *testing.T) {
	footer := testFooter()
	for _, want := range []int{0, 1, 2, -1, 0} {
		footer = footer.Update(tab)
		if got := footer.Focused(); got != want {
			t.Fatalf("Tab: focus on %d, want %d", got, want)
		}
	}
	for _, want := range []int{-1, 2} {
		footer = footer.Update(shiftTab)
		if got := footer.Focused(); got != want {
			t.Fatalf("Shift-Tab: focus on %d, want %d", got, want)
		}
	}
}

func TestFooterFocusPassesOverADisabledButton(t *testing.T) {
	off := key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "check"), key.WithDisabled())
	footer := NewFooter(off, Keys.Quit).Update(tab)
	if got := footer.Focused(); got != 1 {
		t.Errorf("focus on %d, want 1", got)
	}
}
