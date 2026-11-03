package apply

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/server/roles"
	"github.com/zadykian/ws/internal/server/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// Config is what Run needs.
type Config struct {
	Options
	Values values.Values
	// Secrets are the decrypted secrets, for the roles that read them.
	Secrets secrets.Secrets
	M       effect.Machine
	Out     effect.Out
	// Prepare are the steps that come first, whatever roles run: what every role needs.
	Prepare []step.Step
}

// Run runs Prepare, then the steps of each role in order, and returns the roles that changed, or
// that would under Check.
func Run(ctx context.Context, c Config, list []roles.Role) ([]string, error) {
	if _, err := step.Run(ctx, c.Prepare, c.Out, c.Check); err != nil {
		return nil, err
	}
	var changed []string
	for _, r := range list {
		did, err := step.Run(ctx, r.Steps(c.Values, c.Secrets, c.M, c.Out), c.Out, c.Check)
		if err != nil {
			return changed, fmt.Errorf("%s: %w", r.Name, err)
		}
		if did {
			changed = append(changed, r.Name)
		}
	}
	return changed, nil
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
