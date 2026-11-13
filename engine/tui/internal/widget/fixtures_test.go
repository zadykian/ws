package widget

import "github.com/zadykian/ws/engine/tui/internal/tuitest"

// The keys and fixtures that the tests share, under the names these tests use.
var (
	tab      = tuitest.Tab
	shiftTab = tuitest.ShiftTab
	enter    = tuitest.Enter
	left     = tuitest.Left
	right    = tuitest.Right
	space    = tuitest.Space
	press    = tuitest.Press

	stripped = tuitest.Stripped
	noColor  = tuitest.NoColor
	sgr      = tuitest.SGR

	testTheme = NewTheme(func(string) string { return "" })
)
