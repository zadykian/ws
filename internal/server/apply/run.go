package apply

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
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
	// Prepare are the steps that come first while a playbook role is left: ansible-playbook's
	// packages and the playbook itself. They are never a check, as ansible-playbook cannot run
	// without them.
	Prepare []step.Step
	// Dir is the unpacked playbook, with site.yml, inventory.ini, ansible.cfg and collections.
	Dir string
	// Log is the file that ansible-playbook logs to, for its recap. Each run truncates it.
	Log string
}

// recap is the playbook's recap line of localhost, in its log after a timestamp.
var recap = regexp.MustCompile(`localhost +: ok=\d+ +changed=(\d+) `)

// Run runs the segments in order and returns the roles that changed, or that would under Check.
// The playbook's roles of a segment count as changed together, as its recap names no role.
func Run(ctx context.Context, c Config, segs []Segment) ([]string, error) {
	if hasPlaybook(segs) {
		if _, err := step.Run(ctx, c.Prepare, c.Out, false); err != nil {
			return nil, err
		}
	}
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

func hasPlaybook(segs []Segment) bool {
	for _, s := range segs {
		if s.Playbook != nil {
			return true
		}
	}
	return false
}

// runPlaybook runs ansible-playbook as a child, for the roles' tags, and reports whether its recap
// counts a change.
func (c Config) runPlaybook(ctx context.Context, names []string) (bool, error) {
	if err := c.M.WriteFile(c.Log, nil, 0o600); err != nil {
		return false, err
	}
	argv := []string{
		"LC_ALL=C.UTF-8", "ANSIBLE_CONFIG=" + c.Dir + "/ansible.cfg",
		"ANSIBLE_COLLECTIONS_PATH=" + c.Dir + "/collections", "ANSIBLE_LOG_PATH=" + c.Log,
		"ansible-playbook", "-i", c.Dir + "/inventory.ini", c.Dir + "/site.yml", "--tags", strings.Join(names, ","),
	}
	if c.Check {
		argv = append(argv, "--check", "--diff")
	}
	// env sets the variables of the child alone; Pass sends its output to ws's standard error.
	if err := c.M.Pass(ctx, false, "env", append(argv, c.Args...)...); err != nil {
		return false, fmt.Errorf("the playbook for %s: %w", strings.Join(names, ", "), err)
	}
	log, err := c.M.ReadFile(c.Log)
	if err != nil {
		return false, err
	}
	m := recap.FindSubmatch(log)
	if m == nil {
		return false, fmt.Errorf("the playbook for %s: no recap in %s", strings.Join(names, ", "), c.Log)
	}
	n, _ := strconv.Atoi(string(m[1]))
	return n > 0, nil
}
