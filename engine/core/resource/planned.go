package resource

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/engine/core/step"
)

// Planned is a part whose changes are planned already, as a role of the Mac does once it has read
// its files for a run.
type Planned struct {
	Changes []step.Change
}

// Plan returns the changes, each of which must apply itself.
func (p Planned) Plan(context.Context) ([]Change, error) {
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
