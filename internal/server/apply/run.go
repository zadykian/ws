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
	Out     effect.Out
	// Observer takes the events of every step, which name their role.
	Observer step.Observer
	// Prepare are the steps that come first, whatever roles run: what every role needs.
	Prepare []step.Step
}

// Run runs Prepare, then the steps of each role in order, and returns the roles that changed, or
// that would under Check. Given --secrets, the roles take the secrets as the whole truth
// (secrets.Secrets.Whole).
func Run(ctx context.Context, c Config, list []roles.Role) ([]string, error) {
	s := c.Secrets
	s.Whole = c.Options.Secrets != ""
	o := step.Options{Observer: c.Observer, Check: c.Check, Parent: Prepare}
	if _, err := step.RunWith(ctx, c.Prepare, o); err != nil {
		c.unreached(list, s)
		return nil, err
	}
	var changed []string
	for i, r := range list {
		o.Parent = r.Name
		did, err := step.RunWith(ctx, r.Steps(c.Values, s, c.M, c.Out), o)
		if err != nil {
			c.unreached(list[i+1:], s)
			return changed, fmt.Errorf("%s: %w", r.Name, err)
		}
		if did {
			changed = append(changed, r.Name)
		}
	}
	return changed, nil
}

// unreached ends the steps of the roles that a failure stopped short of.
func (c Config) unreached(list []roles.Role, s secrets.Secrets) {
	for _, r := range list {
		o := step.Options{Observer: c.Observer, Check: c.Check, Parent: r.Name}
		step.Unreached(r.Steps(c.Values, s, c.M, c.Out), o)
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
