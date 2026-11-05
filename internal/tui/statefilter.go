package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
)

// filterPrompt starts the filter's line, with the key that opens it.
const filterPrompt = "/ "

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
