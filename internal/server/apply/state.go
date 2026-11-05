package apply

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/roles"
	"github.com/zadykian/ws/step"
)

// RoleState is what a role's steps report, as ws server state prints it. Prepare's steps stand
// under the role Prepare.
type RoleState struct {
	Role  string        `json:"role"`
	Steps []step.Report `json:"steps"`
}

// State reads what Prepare's steps and then each role's steps report, and applies none. It goes
// past a step that fails, which stands in its report. Only the end of ctx fails it, and it returns
// the roles it has read. The roles get no secrets: a step that needs them would write them out to
// plan, as ikeward's install does, and a report has no use for them.
func State(ctx context.Context, c Config, list []roles.Role) ([]RoleState, error) {
	s := secrets.Secrets{Unread: true}
	var states []RoleState
	read := func(name string, steps []step.Step) error {
		reports, err := step.Read(ctx, steps)
		states = append(states, RoleState{Role: name, Steps: reports})
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		return nil
	}
	if err := read(Prepare, c.Prepare); err != nil {
		return states, err
	}
	for _, r := range list {
		if err := read(r.Name, r.Steps(c.Values, s, c.M, c.Out)); err != nil {
			return states, err
		}
	}
	return states, nil
}
