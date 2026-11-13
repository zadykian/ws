package bridge

import (
	"context"
	"fmt"
	"net/netip"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/systemd"
)

// Check stops the run where docker0 lacks the address the values give. docker0 takes a moved
// address only as dockerd starts with no container running. daemon.json's dns, resolved and ufw
// take it at once, and lookups to it get no answers until docker0 moves. A first install passes,
// as no container runs yet. A check shows the move as a change instead.
type Check struct {
	M    effect.Machine
	Out  effect.Out
	Held func() ([]netip.Prefix, error)
	Want string // the bridge's address with its prefix length
}

type bridgeMoves struct{ holds, want string }

func (c bridgeMoves) String() string {
	return fmt.Sprintf("docker0 holds %s, and would take %s as Docker starts with no container running", c.holds, c.want)
}

// Apply fails: the run cannot move docker0.
func (c bridgeMoves) Apply(context.Context) error {
	return fmt.Errorf("docker0 holds %s, not %s: Docker moves it only as it starts with no container running, "+
		"and until then lookups to a moved address get no answers. Reboot, or stop every container and restart "+
		"Docker, then run ws again", c.holds, c.want)
}

// Plan reads docker0's addresses.
func (c Check) Plan(context.Context) ([]resource.Change, error) {
	if !systemd.Booted(c.M) {
		c.Out.Say("docker0: skipped, systemd is not running")
		return nil, nil
	}
	held, err := c.Held()
	if err != nil {
		return nil, err
	}
	var text []string
	for _, p := range held {
		if p.String() == c.Want {
			return nil, nil
		}
		text = append(text, p.String())
	}
	holds := "no address"
	if len(text) > 0 {
		holds = strings.Join(text, ", ")
	}
	return []resource.Change{bridgeMoves{holds, c.Want}}, nil
}
