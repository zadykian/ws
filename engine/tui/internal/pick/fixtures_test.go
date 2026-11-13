package pick

import (
	"context"
	"io"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/tui/internal/list"
	"github.com/zadykian/ws/engine/tui/internal/run"
	"github.com/zadykian/ws/engine/tui/internal/tuitest"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// The keys and fixtures that the tests share, under the names these tests use.
var (
	enter = tuitest.Enter
	esc   = tuitest.Esc
	press = tuitest.Press

	stripped = tuitest.Stripped
	testCut  = tuitest.Cut
	planned  = tuitest.Planned

	testTheme = widget.NewTheme(func(string) string { return "" })
)

type oneChange = tuitest.OneChange

// noKeys is a Config whose terminal gives no key, and whose output is out.
func noKeys(out io.Writer) run.Config {
	return run.Config{In: strings.NewReader(""), Out: out, Theme: testTheme, Cut: testCut}
}

// waitForCtx is a work that heeds its context.
func waitForCtx(ctx context.Context, _ *run.Bridge) error {
	<-ctx.Done()
	return ctx.Err()
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

const leaveBuffer = tuitest.LeaveBuffer
