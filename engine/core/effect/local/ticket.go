package local

import (
	"context"

	"github.com/zadykian/ws/engine/core/effect"
)

// Validate has sudo ask for its password now, ahead of a run's lines. That is sudo -v, which asks at
// the terminal, so an OSMachine under a screen takes the password by its Ticket instead.
func Validate(ctx context.Context, m effect.Machine) error {
	if o, ok := m.(OSMachine); ok && o.Ticket != nil {
		return o.Ticket()
	}
	return m.Pass(ctx, false, "sudo", "-v")
}
