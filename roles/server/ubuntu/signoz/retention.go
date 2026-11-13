package signoz

import (
	"github.com/zadykian/ws/engine/core/effect"

	"github.com/zadykian/ws/roles/server/ubuntu/signoz/internal/api"
)

// Retention is the step that keeps SigNoz's data for Days. SigNoz keeps traces and logs for 15 days
// and metrics for 30, unless told otherwise. A new retention applies only to data that comes in
// after it, so the step sets it before the collector sends anything.
type Retention struct {
	M    effect.Machine
	Out  effect.Out
	API  api.Client
	Days int
	// Migrator is how long Plan waits for the schema migrator: 120 tries, 5 seconds apart, if zero.
	Migrator api.Wait
	// Altered is how long Apply waits for SigNoz to alter the tables of traces and metrics: 60
	// tries, 5 seconds apart, if zero.
	Altered api.Wait
}

// Name returns the step's name.
func (Retention) Name() string { return "signoz retention" }

// WaitsOn returns the step before.
func (Retention) WaitsOn() string { return "signoz ready" }
