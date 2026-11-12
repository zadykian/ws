package apply

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/roles"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// Prepare is the parent of Config.Prepare's steps in a run's events, as a role is of its own.
const Prepare = "prepare"

// Config is what Run needs.
type Config struct {
	Options
	Values values.Values
	// Secrets are the decrypted secrets, for the roles that read them.
	Secrets secrets.Secrets
	M       effect.Machine
	// U are the units every role of the run gets; prepareServer sets it once.
	U   effect.Units
	Out effect.Out
	// Observer takes the events of every step, which name their role.
	Observer step.Observer
	// Prepare are the steps that come first, whatever roles run: what every role needs.
	Prepare []step.Step
	// Stop is the Stop of every step.Options here. Nil never stops.
	Stop func() error
}

// Steps returns r's steps as Run runs them, on M and Out. Given --secrets, they take the secrets as
// the whole truth (secrets.Secrets.Whole).
func (c Config) Steps(r roles.Role) []step.Step {
	s := c.Secrets
	s.Whole = c.Options.Secrets != ""
	return r.Steps(c.Values, s, c.M, c.U, c.Out)
}

// Run runs Prepare, then the steps of each role in order, and returns the roles that changed, or
// that would under Check. Only leaves steps out of the roles alone, as Prepare serves every step.
func Run(ctx context.Context, c Config, list []roles.Role) ([]string, error) {
	o := step.Options{Observer: c.Observer, Check: c.Check, Parent: Prepare, Stop: c.Stop}
	if _, err := step.RunWith(ctx, c.Prepare, o); err != nil {
		c.unreached(list)
		return nil, err
	}
	o.LeftOut = c.leftOut()
	var changed []string
	for i, r := range list {
		o.Parent = r.Name
		did, err := step.RunWith(ctx, c.Steps(r), o)
		if err != nil {
			c.unreached(list[i+1:])
			return changed, fmt.Errorf("%s: %w", r.Name, err)
		}
		if did {
			changed = append(changed, r.Name)
		}
	}
	return changed, nil
}

// unreached ends the steps of the roles that a failure stopped short of.
func (c Config) unreached(list []roles.Role) {
	for _, r := range list {
		o := step.Options{Observer: c.Observer, Check: c.Check, Parent: r.Name}
		step.Unreached(c.Steps(r), o)
	}
}

// Summary is ws's last line: the roles that changed, or that --check would change, and none on a
// second run.
func Summary(changed []string, check bool) string {
	verb := "changed"
	if check {
		verb = "would change"
	}
	if len(changed) == 0 {
		return verb + ": none"
	}
	return verb + ": " + strings.Join(changed, ", ")
}
