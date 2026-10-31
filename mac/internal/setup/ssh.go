package setup

import (
	"context"
	"time"

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
)

// SSH writes ~/.ssh/config.d/ws and puts its Include first in ~/.ssh/config. It stops where
// config.d/ws is a file it did not write, where the Include could not go first, where the key that
// ssh offers ws is wrong, and where ssh refuses the new file, alone or as ssh reads it whole.
type SSH struct {
	Conf conf.Conf
	Home string
	M    effect.Machine
	Out  effect.Out
	Now  func() time.Time
	// AssumeTunnel has ws go through the tunnel wherever mac.conf has WS_PUBLIC_KEY, as under --check,
	// which does not load the daemon.
	AssumeTunnel bool
	// Again is the second run of the step, which the tunnel and cld may have changed ssh's file for.
	Again bool
}

// Name is ssh, or ssh again.
func (s SSH) Name() string {
	if s.Again {
		return "ssh again"
	}
	return "ssh"
}

// WaitsOn is the tunnel for the second run.
func (s SSH) WaitsOn() string {
	if s.Again {
		return "tunnel"
	}
	return ""
}

// Apply applies the changes in order.
func (s SSH) Apply(ctx context.Context, p step.Plan) error {
	for _, c := range p.Changes {
		if err := Apply(ctx, s.M, s.Out, c); err != nil {
			return err
		}
	}
	return nil
}
