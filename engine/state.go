package engine

import "context"

// State reports what a Target's roles hold, as text or as JSON.
func State(ctx context.Context, c Call, t Target) error {
	report, err := t.State(ctx, c)
	if err != nil {
		return err
	}
	return report.Print(ctx, c)
}
