package queue

import (
	"context"
	"io"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/tui/internal/run"
	"github.com/zadykian/ws/engine/tui/internal/tuitest"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// The keys and fixtures that the tests share, under the names these tests use.
var (
	tab      = tuitest.Tab
	shiftTab = tuitest.ShiftTab
	enter    = tuitest.Enter
	pgUp     = tuitest.PgUp
	press    = tuitest.Press

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
