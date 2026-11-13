package tui

import "github.com/zadykian/ws/engine/tui/internal/tuitest"

// The keys and fixtures that the tests of every package share, under the names these tests use.
var (
	tab      = tuitest.Tab
	shiftTab = tuitest.ShiftTab
	enter    = tuitest.Enter
	left     = tuitest.Left
	right    = tuitest.Right
	space    = tuitest.Space
	esc      = tuitest.Esc
	ctrlC    = tuitest.CtrlC
	pgUp     = tuitest.PgUp
	down     = tuitest.Down
	press    = tuitest.Press

	stripped = tuitest.Stripped
	noColor  = tuitest.NoColor
	sgr      = tuitest.SGR
	testCut  = tuitest.Cut
	planned  = tuitest.Planned

	window    = Size{80, 24}
	testTheme = NewTheme(func(string) string { return "" })
)

const leaveBuffer = tuitest.LeaveBuffer

type oneChange = tuitest.OneChange
