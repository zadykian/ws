package tui

import (
	"fmt"

	"github.com/zadykian/ws/engine/tui/internal/tuitest"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// The keys and fixtures that the tests of every package share, under the names these tests use.
var (
	tab      = tuitest.Tab
	shiftTab = tuitest.ShiftTab
	enter    = tuitest.Enter
	esc      = tuitest.Esc
	ctrlC    = tuitest.CtrlC
	pgUp     = tuitest.PgUp
	down     = tuitest.Down
	press    = tuitest.Press

	stripped = tuitest.Stripped
	noColor  = tuitest.NoColor
	testCut  = tuitest.Cut
	planned  = tuitest.Planned

	window    = widget.Size{Width: 80, Height: 24}
	testTheme = widget.NewTheme(func(string) string { return "" })
)

const leaveBuffer = tuitest.LeaveBuffer

type oneChange = tuitest.OneChange

// waySpecs asks for a port where the host has a value, as a wizard's question hangs on the one above
// it.
func waySpecs(edits map[string]string) []widget.FieldSpec {
	if edits["HOST"] == "none" {
		return []widget.FieldSpec{{Label: "HOST", Value: "ws.example"}, {Label: "USER"}}
	}
	return []widget.FieldSpec{{Label: "HOST", Value: "ws.example"}, {Label: "PORT", Default: "22"}, {Label: "USER"}}
}

// longSpecs is more fields than a short window holds.
func longSpecs(map[string]string) []widget.FieldSpec {
	specs := make([]widget.FieldSpec, 0, 6)
	for i := range 6 {
		specs = append(specs, widget.FieldSpec{Label: fmt.Sprintf("KEY%d", i), Help: "says more"})
	}
	return specs
}
