package tui

import (
	"io"
	"strings"

	"github.com/zadykian/ws/engine/tui/internal/tuitest"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// The keys and fixtures that the tests of every package share, under the names these tests use.
var (
	tab      = tuitest.Tab
	shiftTab = tuitest.ShiftTab
	enter    = tuitest.Enter
	esc      = tuitest.Esc
	down     = tuitest.Down
	press    = tuitest.Press

	stripped = tuitest.Stripped
	noColor  = tuitest.NoColor
	testCut  = tuitest.Cut

	testTheme = widget.NewTheme(func(string) string { return "" })
)

const leaveBuffer = tuitest.LeaveBuffer

// noKeys is a Config whose terminal gives no key, and whose output is out.
func noKeys(out io.Writer) Config {
	return Config{In: strings.NewReader(""), Out: out, Theme: testTheme, Cut: testCut}
}
