package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// shown returns the rows of tab that the filter leaves. A row is its name to the filter, so that a
// word finds what the list shows.
func (s StateScreen) shown(tab int) []stateRow {
	words := strings.ToLower(s.filter.Words())
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

// edited gives the filter a key or a paste, and the list follows each one.
func (s StateScreen) edited(msg tea.Msg) StateScreen {
	picks := s.picks()
	s.filter = s.filter.Edited(msg)
	s.scroll = 0
	return s.refound(picks)
}

// cleared drops the filter's words.
func (s StateScreen) cleared() StateScreen {
	picks := s.picks()
	s.filter = s.filter.Cleared()
	return s.refound(picks)
}
