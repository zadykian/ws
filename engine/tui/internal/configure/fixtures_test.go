package configure

import (
	"fmt"
	"io"
	"strings"

	"github.com/zadykian/ws/engine/tui/internal/run"
	"github.com/zadykian/ws/engine/tui/internal/tuitest"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// The keys and fixtures that the tests share, under the names these tests use.
var (
	tab      = tuitest.Tab
	shiftTab = tuitest.ShiftTab
	enter    = tuitest.Enter
	esc      = tuitest.Esc
	press    = tuitest.Press

	stripped = tuitest.Stripped
	testCut  = tuitest.Cut

	testTheme = widget.NewTheme(func(string) string { return "" })
)

// noKeys is a Config whose terminal gives no key, and whose output is out.
func noKeys(out io.Writer) run.Config {
	return run.Config{In: strings.NewReader(""), Out: out, Theme: testTheme, Cut: testCut}
}

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
