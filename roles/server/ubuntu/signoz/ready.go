package signoz

import (
	"context"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/signoz/internal/api"
)

// Ready is the step that waits for SigNoz's API to answer. The API answers only where Docker's
// daemon runs SigNoz, so without systemd booted the step skips.
type Ready struct {
	M   effect.Machine
	Out effect.Out
	API api.Client
	// Wait is 60 tries, 5 seconds apart, if zero.
	Wait api.Wait
}

type await struct{ r Ready }

func (await) String() string { return "wait for SigNoz to answer" }

func (c await) Apply(ctx context.Context) error {
	w := c.r.Wait.OrDefault(60, 5*time.Second)
	return w.Poll(ctx, "SigNoz's readyz", func() (bool, string, error) {
		if err := c.r.API.Get(ctx, "/api/v2/readyz", nil); err != nil {
			return false, err.Error(), nil
		}
		return true, "", nil
	})
}

// Name returns the step's name.
func (Ready) Name() string { return "signoz ready" }

// WaitsOn returns the step before.
func (Ready) WaitsOn() string { return "signoz up" }

// Plan asks once.
func (r Ready) Plan(ctx context.Context) (step.Plan, error) {
	if !host.Booted(r.M) {
		r.Out.Say("%s: skipped, systemd is not running", r.Name())
		return step.Plan{}, nil
	}
	if r.API.Get(ctx, "/api/v2/readyz", nil) == nil {
		return step.Plan{}, nil
	}
	return step.Plan{Changes: []step.Change{await{r}}}, nil
}

// Apply waits for the answer.
func (Ready) Apply(ctx context.Context, p step.Plan) error { return p.Changes[0].(await).Apply(ctx) }
