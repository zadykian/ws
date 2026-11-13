package tui

import (
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
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
		footer, _ = footer.Update(tab)
		if got := footer.Focused(); got != want {
			t.Fatalf("Tab: focus on %d, want %d", got, want)
		}
	}
	for _, want := range []int{-1, 2} {
		footer, _ = footer.Update(shiftTab)
		if got := footer.Focused(); got != want {
			t.Fatalf("Shift-Tab: focus on %d, want %d", got, want)
		}
	}
}

func TestFooterPress(t *testing.T) {
	onCheck, _ := testFooter().Update(tab)
	onCheck, _ = onCheck.Update(tab)
	cases := []struct {
		name   string
		footer Footer
		msg    tea.KeyPressMsg
		want   int
	}{
		{"a shortcut", testFooter(), press('c'), 1},
		{"a key of no button", testFooter(), press('x'), -1},
		{"Enter as a shortcut, the panes in focus", testFooter(), enter, 0},
		{"Enter on the focused button, over the shortcut", onCheck, enter, 1},
		{"a shortcut, another button in focus", onCheck, press('q'), 2},
		{"the shortcut of a disabled button", testFooter().Enable(1, false), press('c'), -1},
	}
	for _, c := range cases {
		if _, got := c.footer.Update(c.msg); got != c.want {
			t.Errorf("%s: pressed %d, want %d", c.name, got, c.want)
		}
	}
}

func TestFooterFocusPassesOverADisabledButton(t *testing.T) {
	off := key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "check"), key.WithDisabled())
	footer, _ := NewFooter(off, Keys.Quit).Update(tab)
	if got := footer.Focused(); got != 1 {
		t.Errorf("focus on %d, want 1", got)
	}
}

func TestFooterEnableMovesTheFocusOffAButtonTurnedOff(t *testing.T) {
	before, _ := testFooter().Update(tab)
	after := before.Enable(0, false)
	if got := after.Focused(); got != 1 {
		t.Errorf("focus on %d, want 1", got)
	}
	if !before.buttons[0].Enabled() {
		t.Error("Enable changed the older value's buttons")
	}
}

func TestFooterRendersItsShortcutsInRowsThatFit(t *testing.T) {
	for width, want := range map[int]string{
		80: "[enter] run  [c] check  [q] quit",
		22: "[enter] run  [c] check\n[q] quit",
		10: "[enter] run\n[c] check\n[q] quit",
	} {
		if got := stripped(testFooter().render(NewTheme(noColor("1")), width)); got != want {
			t.Errorf("%d columns: %q, want %q", width, got, want)
		}
	}
}

func TestFooterMarksItsFocusWithoutColor(t *testing.T) {
	theme := NewTheme(noColor("1"))
	rest := testFooter().render(theme, 80)
	focused, _ := testFooter().Update(tab)
	if focused.render(theme, 80) == rest {
		t.Error("a focused button reads as any other")
	}
}
