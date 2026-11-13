package tui

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/tui/internal/list"
	"github.com/zadykian/ws/engine/tui/internal/tuitest"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// The keys and fixtures that the tests of every package share, under the names these tests use.
var (
	tab      = tuitest.Tab
	shiftTab = tuitest.ShiftTab
	enter    = tuitest.Enter
	esc      = tuitest.Esc
	pgUp     = tuitest.PgUp
	down     = tuitest.Down
	press    = tuitest.Press

	stripped = tuitest.Stripped
	noColor  = tuitest.NoColor
	testCut  = tuitest.Cut
	planned  = tuitest.Planned

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

// chain is a role whose steps wait on each other, beside one that waits on nothing.
func chain() list.PickList {
	waits := map[string]string{"two": "one", "three": "two"}
	return list.NewPickList([]list.Group{{Name: "base", Steps: []string{"one", "two", "three", "alone"}, Waits: waits}})
}

// choosing returns l with the named steps' own boxes ticked.
func choosing(l list.PickList, names ...string) list.PickList {
	l.Rows = slices.Clone(l.Rows)
	for i := range l.Rows {
		l.Rows[i].Chosen = slices.Contains(names, l.Rows[i].Name)
	}
	return l.Settled()
}

// cursorOn returns the cursor on the step of that name.
func cursorOn(l list.PickList, name string) widget.Cursor {
	return widget.Cursor(slices.IndexFunc(l.Lines(), func(line list.PickLine) bool {
		return !line.Heading && l.Rows[line.Row].Name == name
	}))
}

// threeRoles has changes in the first two.
func threeRoles() list.PickList {
	l := list.NewPickList([]list.Group{{Name: "a", Steps: []string{"one"}}, {Name: "b", Steps: []string{"two", "three"}}, {Name: "c", Steps: []string{"four"}}})
	return l.Take(planned("a", "one", 0, oneChange("x"))).Take(planned("b", "two", 0, oneChange("x")))
}

// noKeys is a Config whose terminal gives no key, and whose output is out.
func noKeys(out io.Writer) Config {
	return Config{In: strings.NewReader(""), Out: out, Theme: testTheme, Cut: testCut}
}

// waitForCtx is a work that heeds its context.
func waitForCtx(ctx context.Context, _ *Bridge) error {
	<-ctx.Done()
	return ctx.Err()
}

// runs says whether cmd, a batch or not, gives want.
func runs(cmd tea.Cmd, want tea.Msg) bool {
	if cmd == nil {
		return false
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		return slices.ContainsFunc(batch, func(c tea.Cmd) bool { return runs(c, want) })
	}
	return msg == want
}
