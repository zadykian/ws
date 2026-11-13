package resource

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/engine/core/step"
)

// Planned is a part whose changes are planned already, as a role of the Mac does once it has read
// its files for a run. Seen reports how the same files stand, and Err stops the plan, so that a
// report of the role goes on past what a plan refuses.
type Planned struct {
	Changes []step.Change
	Seen    step.Reporter
	Err     error
}

// Plan returns the changes, each of which must apply itself, or Err.
func (p Planned) Plan(context.Context) ([]Change, error) {
	if p.Err != nil {
		return nil, p.Err
	}
	changes := make([]Change, len(p.Changes))
	for i, c := range p.Changes {
		applying, ok := c.(Change)
		if !ok {
			return nil, fmt.Errorf("%s does not apply itself", step.Headline(c))
		}
		changes[i] = applying
	}
	return changes, nil
}

// Report reports what Seen does, or nothing.
func (p Planned) Report(ctx context.Context) ([]step.Item, error) {
	if p.Seen == nil {
		return nil, nil
	}
	return p.Seen.Report(ctx)
}
