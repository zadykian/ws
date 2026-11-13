package api

import (
	"context"
	"fmt"
	"time"
)

// Wait is how long and how often a step waits for what SigNoz does on its own.
type Wait struct {
	Tries int
	Delay time.Duration
	// Sleep waits for Delay, or for the context. Nil sleeps for real.
	Sleep func(ctx context.Context, d time.Duration) error
}

// OrDefault returns w, or the default where w has no tries.
func (w Wait) OrDefault(tries int, delay time.Duration) Wait {
	if w.Tries == 0 {
		w.Tries, w.Delay = tries, delay
	}
	return w
}

func (w Wait) sleep(ctx context.Context) error {
	if w.Sleep != nil {
		return w.Sleep(ctx, w.Delay)
	}
	t := time.NewTimer(w.Delay)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// Check reports whether the wait is over, and if not, why.
type Check func() (done bool, why string, err error)

// Poll calls c up to Tries times, a Delay apart, and returns when it reports done or fails.
// It fails with what the last call said, if c never reports done.
func (w Wait) Poll(ctx context.Context, what string, c Check) error {
	var why string
	for i := range w.Tries {
		if i > 0 {
			if err := w.sleep(ctx); err != nil {
				return err
			}
		}
		done, said, err := c()
		if err != nil {
			return err
		}
		if done {
			return nil
		}
		why = said
	}
	return fmt.Errorf("%s: gave up after %d tries, last: %s", what, w.Tries, why)
}
