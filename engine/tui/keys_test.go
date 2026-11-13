package tui

import (
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

func TestKeysMatchWhatTheTerminalSends(t *testing.T) {
	cases := []struct {
		name    string
		msg     tea.KeyPressMsg
		binding key.Binding
	}{
		{"tab", tea.KeyPressMsg{Code: tea.KeyTab}, Keys.Next},
		{"shift+tab", tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}, Keys.Prev},
		{"enter", tea.KeyPressMsg{Code: tea.KeyEnter}, Keys.Press},
		{"up", tea.KeyPressMsg{Code: tea.KeyUp}, Keys.Up},
		{"j", press('j'), Keys.Down},
		{"page up", tea.KeyPressMsg{Code: tea.KeyPgUp}, Keys.PageUp},
		{"ctrl+u", tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}, Keys.PageUp},
		{"ctrl+d", tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl}, Keys.PageDown},
		{"home", tea.KeyPressMsg{Code: tea.KeyHome}, Keys.Top},
		{"G", tea.KeyPressMsg{Code: 'g', Mod: tea.ModShift, Text: "G"}, Keys.Bottom},
		{"q", press('q'), Keys.Quit},
		{"ctrl+c", tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}, Keys.Interrupt},
	}
	for _, c := range cases {
		if !key.Matches(c.msg, c.binding) {
			t.Errorf("%s: %q matches no binding", c.name, c.msg.String())
		}
	}
}
