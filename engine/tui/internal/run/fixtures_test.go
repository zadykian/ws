package run

import (
	"github.com/zadykian/ws/engine/tui/internal/tuitest"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// The keys and fixtures that the tests share, under the names these tests use.
var (
	tab     = tuitest.Tab
	enter   = tuitest.Enter
	esc     = tuitest.Esc
	ctrlC   = tuitest.CtrlC
	pgUp    = tuitest.PgUp
	press   = tuitest.Press
	testCut = tuitest.Cut

	stripped = tuitest.Stripped

	window    = widget.Size{Width: 80, Height: 24}
	testTheme = widget.NewTheme(func(string) string { return "" })
)

const leaveBuffer = tuitest.LeaveBuffer

type oneChange = tuitest.OneChange
