package apply

import (
	"context"
	"fmt"
	"strings"

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
	// Dir is the unpacked playbook, with site.yml, inventory.ini, ansible.cfg and collections.
	Dir string
}

// Run runs the segments in order and returns the roles that changed, or that would under Check.
// The playbook's roles of a segment count as changed together, as its recap names no role.
func Run(ctx context.Context, c Config, segs []Segment) ([]string, error) {
	var changed []string
	for _, s := range segs {
		if s.Playbook == nil {
			did, err := step.Run(ctx, s.Role.Steps(c.Values, c.M, c.Out), c.Out, c.Check)
			if err != nil {
				return changed, fmt.Errorf("%s: %w", s.Role.Name, err)
			}
			if did {
				changed = append(changed, s.Role.Name)
			}
			continue
		}
		did, err := c.runPlaybook(ctx, s.Playbook)
		if err != nil {
			return changed, err
		}
		if did {
			changed = append(changed, s.Playbook...)
		}
	}
	return changed, nil
}

// runPlaybook runs ansible-playbook as a child, for the roles' tags. It reports a change at once, as
// nothing reads its recap yet.
func (c Config) runPlaybook(ctx context.Context, names []string) (bool, error) {
	argv := []string{
		"LC_ALL=C.UTF-8", "ANSIBLE_CONFIG=" + c.Dir + "/ansible.cfg",
		"ANSIBLE_COLLECTIONS_PATH=" + c.Dir + "/collections",
		"ansible-playbook", "-i", c.Dir + "/inventory.ini", c.Dir + "/site.yml", "--tags", strings.Join(names, ","),
	}
	if c.Check {
		argv = append(argv, "--check", "--diff")
	}
	// env sets the variables of the child alone; Pass sends its output to ws's standard error.
	if err := c.M.Pass(ctx, false, "env", append(argv, c.Args...)...); err != nil {
		return false, fmt.Errorf("the playbook for %s: %w", strings.Join(names, ", "), err)
	}
	return true, nil
}
