package engine

import (
	"context"

	"github.com/zadykian/ws/engine/tui"
)

// State reports what a Target's roles hold. At a terminal it draws the state screen, and anywhere
// else it prints the report, as text or as JSON.
func State(ctx context.Context, c Call, t Target) error {
	report, err := t.State(ctx, c)
	if err != nil {
		return err
	}
	if DrawsReport(c) {
		if report.M != nil {
			c.Machine = report.M // ws_home confines it.
		}
		return tui.RunState(ctx, ScreenConfig(c), c.Path, report.Read)
	}
	return report.Print(ctx, c)
}
