package apply

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// Config is what Run needs.
type Config struct {
	Options
	Values values.Values
	M      effect.Machine
	Out    effect.Out
}

// Run runs the segments in order and returns the roles that changed, or that would under Check.
func Run(ctx context.Context, c Config, segs []Segment) ([]string, error) {
	var changed []string
	for _, s := range segs {
		if s.Playbook != nil {
			return changed, fmt.Errorf("the playbook for %v: not run yet", s.Playbook)
		}
		did, err := step.Run(ctx, s.Role.Steps(c.Values, c.M, c.Out), c.Out, c.Check)
		if err != nil {
			return changed, fmt.Errorf("%s: %w", s.Role.Name, err)
		}
		if did {
			changed = append(changed, s.Role.Name)
		}
	}
	return changed, nil
}
