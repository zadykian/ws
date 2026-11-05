package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

var (
	left  = tea.KeyPressMsg{Code: tea.KeyLeft}
	right = tea.KeyPressMsg{Code: tea.KeyRight}
	space = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
)

func choice(fallback string) FormField {
	spec := FieldSpec{Kind: ChoiceField, Options: []string{"lan", "cloud", "mac-vm"}, Default: fallback}
	return NewFormField(spec).Focus()
}

func TestChoiceMovesRoundItsOptions(t *testing.T) {
	cases := []struct {
		name string
		keys []tea.Msg
		want string
	}{
		{"right", []tea.Msg{right}, "cloud"},
		{"space", []tea.Msg{space}, "cloud"},
		{"past the last", []tea.Msg{right, right, right}, "lan"},
		{"before the first", []tea.Msg{left}, "mac-vm"},
		{"h and l", []tea.Msg{press('l'), press('l'), press('h')}, "cloud"},
		{"another key", []tea.Msg{press('x'), tea.PasteMsg{Content: "cloud"}}, "lan"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := choice("")
			for _, msg := range c.keys {
				f = f.Update(msg)
			}
			if got := f.Value(); got != c.want {
				t.Errorf("value = %q, want %q", got, c.want)
			}
		})
	}
}
