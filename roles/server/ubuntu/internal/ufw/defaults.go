package ufw

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/systemd"
)

// DefaultFile holds the default policies.
const DefaultFile = "/etc/default/ufw"

// policyWord is how DefaultFile spells a policy.
var policyWord = map[string]string{"allow": "ACCEPT", "deny": "DROP", "reject": "REJECT"}

// Defaults sets ufw's default policies, read from DefaultFile. Policies are allow, deny or
// reject; an empty one is left alone. It skips where systemd is not running, as in CI's container.
type Defaults struct {
	M                  effect.Machine
	Out                effect.Out
	Incoming, Outgoing string
}

type setDefault struct {
	m         effect.Machine
	policy    string
	direction string
}

func (c setDefault) String() string { return "ufw default " + c.policy + " " + c.direction }

func (c setDefault) Apply(ctx context.Context) error {
	_, err := c.m.RunRoot(ctx, "ufw", "default", c.policy, c.direction)
	return err
}

// Plan sets each policy that DefaultFile does not hold.
func (d Defaults) Plan(context.Context) ([]host.Change, error) {
	if !systemd.Booted(d.M) {
		d.Out.Say("ufw defaults: skipped, systemd is not running")
		return nil, nil
	}
	conf, err := readConf(d.M, DefaultFile)
	if err != nil {
		return nil, err
	}
	var changes []host.Change
	for _, p := range []struct{ policy, direction, key string }{
		{d.Incoming, "incoming", "DEFAULT_INPUT_POLICY"},
		{d.Outgoing, "outgoing", "DEFAULT_OUTPUT_POLICY"},
	} {
		if p.policy == "" {
			continue
		}
		word, ok := policyWord[p.policy]
		if !ok {
			return nil, fmt.Errorf("ufw default %s: %q is not allow, deny or reject", p.direction, p.policy)
		}
		if conf[p.key] != word {
			changes = append(changes, setDefault{d.M, p.policy, p.direction})
		}
	}
	return changes, nil
}
