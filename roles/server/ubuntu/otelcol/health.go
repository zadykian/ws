package otelcol

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/systemd"
)

// healthURL is the collector's health_check extension.
const healthURL = "http://127.0.0.1:13133/"

// healthTries and healthWait bound the check: 30 tries, 2 seconds apart.
const (
	healthTries = 30
	healthWait  = 2 * time.Second
)

// answer checks that the collector answers its health check. systemctl returns once the
// collector runs, before its receivers start. One that cannot, such as OTLP's on a port another
// process holds, stops the collector, which then never answers. Without systemd booted it says so and
// plans nothing.
type answer struct {
	m     effect.Machine
	out   effect.Out
	url   string
	tries int
	wait  time.Duration
}

type await struct{ a answer }

func (c await) String() string { return "wait for the collector to answer at " + c.a.url }

func (c await) Apply(ctx context.Context) error {
	var err error
	for try := 1; try <= c.a.tries; try++ {
		if err = c.a.get(ctx); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(c.a.wait):
		}
	}
	return fmt.Errorf("the collector does not answer after %d tries: %w", c.a.tries, err)
}

func (a answer) get(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s answers %s", a.url, resp.Status)
	}
	return nil
}

// Plan plans the wait where the collector does not answer now.
func (a answer) Plan(ctx context.Context) ([]resource.Change, error) {
	if !systemd.Booted(a.m) {
		a.out.Say("collector health: skipped, systemd is not running")
		return nil, nil
	}
	if a.get(ctx) == nil {
		return nil, nil
	}
	return []resource.Change{await{a}}, nil
}
