package ikeward

import (
	"context"
	"fmt"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/systemd"
)

// Later roles may need the VPN, as for a clone from the remote network, so one that is not up
// within about 2 minutes fails the run. The wait is 12 tries, 10 seconds apart.
const (
	StatusTries = 12
	StatusWait  = 10 * time.Second
)

// Status checks that ikeward status passes. Without systemd booted it says so and plans nothing.
// So it does where status fails in a run without the secrets, as InRun says. No unit of that run
// installs the VPN, so a wait would only fail the run 2 minutes later.
type Status struct {
	M     effect.Machine
	Out   effect.Out
	Tries int
	Wait  time.Duration
	InRun bool
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
func (s Status) Plan(ctx context.Context) ([]resource.Change, error) {
	if !systemd.Booted(s.M) {
		s.Out.Say("ikeward status: skipped, systemd is not running")
		return nil, nil
	}
	if _, err := s.M.Run(ctx, binary, "status"); err == nil {
		return nil, nil
	}
	if !s.InRun {
		s.Out.Say("ikeward status: fails, and the run holds no secrets to install the VPN, so it waits for none")
		return nil, nil
	}
	return []resource.Change{awaitVPN{s}}, nil
}
