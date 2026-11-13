// Package tuitest holds the keys and fixtures that the tests of engine/tui's packages share. It
// names none of those packages' types, so that each of them can import it.
package tuitest

import tea "charm.land/bubbletea/v2"

var (
	// Tab moves to the next button.
	Tab = tea.KeyPressMsg{Code: tea.KeyTab}
	// ShiftTab moves to the previous button.
	ShiftTab = tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	// Enter presses a button.
	Enter = tea.KeyPressMsg{Code: tea.KeyEnter}

	// Left moves a choice field's value back.
	Left = tea.KeyPressMsg{Code: tea.KeyLeft}
	// Right moves a choice field's value forward.
	Right = tea.KeyPressMsg{Code: tea.KeyRight}
	// Space ticks a box or picks a choice.
	Space = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}

	// Esc stops a screen.
	Esc = tea.KeyPressMsg{Code: tea.KeyEscape}
	// CtrlC stops a run.
	CtrlC = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
	// PgUp scrolls a pane up.
	PgUp = tea.KeyPressMsg{Code: tea.KeyPgUp}

	// Down is the j key, which moves a cursor down a row.
	Down = Press('j')
)

// Press is the message of a key with no modifier, by its code.
func Press(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Text: string(code)}
}
