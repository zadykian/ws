package ikeward

import (
	"context"
	"fmt"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/systemd"
)

// Later roles may need the VPN, as for a clone from the remote network, so one that is not up
// within about 2 minutes fails the run. The wait is 12 tries, 10 seconds apart.
const (
	statusTries = 12
	statusWait  = 10 * time.Second
)

// Status checks that ikeward status passes. Without systemd booted it says so and plans nothing.
type Status struct {
	M     effect.Machine
	Out   effect.Out
	Tries int
	Wait  time.Duration
}

type awaitVPN struct{ s Status }

func (awaitVPN) String() string { return "wait for the VPN to come up, as ikeward status says" }

func (c awaitVPN) Apply(ctx context.Context) error {
	var err error
	for try := 1; try <= c.s.Tries; try++ {
		if _, err = c.s.M.Run(ctx, binary, "status"); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(c.s.Wait):
		}
	}
	return fmt.Errorf("the VPN is not up after %d tries: %w", c.s.Tries, err)
}

// Plan plans the wait where ikeward status fails now.
func (s Status) Plan(ctx context.Context) ([]host.Change, error) {
	if !systemd.Booted(s.M) {
		s.Out.Say("ikeward status: skipped, systemd is not running")
		return nil, nil
	}
	if _, err := s.M.Run(ctx, binary, "status"); err == nil {
		return nil, nil
	}
	return []host.Change{awaitVPN{s}}, nil
}
