package tui

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// filterPrompt starts the filter's line, with the key that opens it.
const filterPrompt = "/ "

// clearKey drops the filter.
var clearKey = key.NewBinding(key.WithKeys("esc"))

func newFilter() textinput.Model {
	input := newInput()
	input.Prompt = filterPrompt
	return input
}

// shown returns the rows of tab that the filter leaves. A row is its name to the filter, so that a
// word finds what the list shows.
func (s StateScreen) shown(tab int) []stateRow {
	words := strings.ToLower(s.filter.Value())
	if words == "" {
		return s.rows[tab]
	}
	var rows []stateRow
	for _, row := range s.rows[tab] {
		if strings.Contains(strings.ToLower(row.name), words) {
			rows = append(rows, row)
		}
	}
	return rows
}

// edited gives the filter a key or a paste, and the list follows each one. Enter keeps the words and
// Esc drops them. The input's Cmd goes, as a field's does.
func (s StateScreen) edited(msg tea.Msg) StateScreen {
	if press, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(press, Keys.Press):
			s.editing = false
			return s
		case key.Matches(press, clearKey):
			s.editing = false
			return s.cleared()
		}
	}
	picks := s.picks()
	s.filter, _ = s.filter.Update(msg)
	s.scroll = 0
	return s.refound(picks)
}

// cleared drops the filter's words.
func (s StateScreen) cleared() StateScreen {
	picks := s.picks()
	s.filter.Reset()
	return s.refound(picks)
}

// filterLine draws the filter: the input while it has the keys, the words once they stand, and
// nothing without any.
func (s StateScreen) filterLine() string {
	switch {
	case s.editing:
		return s.filter.View()
	case s.filter.Value() != "":
		return s.theme.Muted.Render(filterPrompt + s.filter.Value())
	}
	return ""
}
