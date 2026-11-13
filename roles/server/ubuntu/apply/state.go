package apply

import (
	"context"
	"fmt"
	"slices"

	"github.com/zadykian/ws/engine/core/step"
	roles "github.com/zadykian/ws/roles/server/ubuntu"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
)

// RoleState is what a role's steps report, as ws server state prints it. Prepare's steps stand
// under the role Prepare.
type RoleState struct {
	Role  string        `json:"role"`
	Steps []step.Report `json:"steps"`
}

// StateDoc is what ws server state --json prints, and what the Mac's wizard reads of it. Version is
// that of the ws that read the steps.
type StateDoc struct {
	Version string      `json:"version"`
	Roles   []RoleState `json:"roles"`
}

// State reads what Prepare's steps and then each role's steps report, and applies none. It goes
// past a step that fails, which stands in its report. Only the end of ctx fails it, and it returns
// the roles it has read. The roles get no secrets: a step that needs them would write them out to
// plan, as ikeward's install does, and a report has no use for them. A role named by ROLE/STEP keeps
// those steps' reports, past a read of the whole role, as a step's report follows the steps it waits
// on. Prepare stands whole, as in a run.
func State(ctx context.Context, c Config, list []roles.Role) ([]RoleState, error) {
	s := secrets.Secrets{Unread: true}
	var states []RoleState
	read := func(name string, steps []step.Step, left func(role, step string) bool) error {
		reports, err := step.Read(ctx, steps)
		if left != nil {
			reports = slices.DeleteFunc(reports, func(r step.Report) bool { return left(name, r.Step) })
		}
		states = append(states, RoleState{Role: name, Steps: reports})
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		return nil
	}
	if err := read(Prepare, c.Prepare, nil); err != nil {
		return states, err
	}
	for _, r := range list {
		if err := read(r.Name, r.Steps(c.Values, s, c.M, c.U, c.Out), c.leftOut()); err != nil {
			return states, err
		}
	}
	return states, nil
}
