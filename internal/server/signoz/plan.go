package signoz

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/step"
)

// Plan reads the retention of traces, metrics and logs, once the schema migrator is done. Without
// systemd booted there is no SigNoz to ask, and it skips.
func (r Retention) Plan(ctx context.Context) (step.Plan, error) {
	if !host.Booted(r.M) {
		r.Out.Say("%s: skipped, systemd is not running", r.Name())
		return step.Plan{}, nil
	}
	if err := awaitMigrator(ctx, r.M, r.Migrator); err != nil {
		return step.Plan{}, err
	}
	var p step.Plan
	var due []ttl
	for _, signal := range []string{"traces", "metrics"} {
		t, err := r.readSignal(ctx, signal)
		if err != nil {
			return step.Plan{}, err
		}
		if t.due(r.Days * 24) {
			due = append(due, t)
		}
	}
	if len(due) > 0 {
		p.Changes = append(p.Changes, setSignals{r, due})
	}
	logs, err := r.readLogs(ctx)
	if err != nil {
		return step.Plan{}, err
	}
	if logs.due(r.Days) {
		p.Changes = append(p.Changes, setLogs{r, logs})
	}
	return p, nil
}

// Apply sets what the plan found due.
func (Retention) Apply(ctx context.Context, p step.Plan) error {
	for _, c := range p.Changes {
		if err := c.(host.Change).Apply(ctx); err != nil {
			return fmt.Errorf("%s: %w", step.Headline(c), err)
		}
	}
	return nil
}
