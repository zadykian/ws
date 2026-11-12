package devtools

import (
	"context"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/host"
)

// GHTelemetry turns gh's telemetry off for User, as gh 2.91.0 and later send it unless told not to.
type GHTelemetry struct {
	M    effect.Machine
	User host.User
}

type telemetryOff struct{ g GHTelemetry }

func (telemetryOff) String() string { return "turn gh's telemetry off" }

func (c telemetryOff) Apply(ctx context.Context) error {
	return c.g.User.Pass(ctx, c.g.M, "env", quietGH("config", "set", "telemetry", "disabled")...)
}

// Plan turns telemetry off where gh's setting is not disabled. A gh that is not installed yet
// answers with an error, which counts as on.
func (g GHTelemetry) Plan(ctx context.Context) ([]host.Change, error) {
	out, err := g.User.Output(ctx, g.M, "env", quietGH("config", "get", "telemetry")...)
	if err == nil && strings.TrimSpace(string(out)) == "disabled" {
		return nil, nil
	}
	return []host.Change{telemetryOff{g}}, nil
}

// quietGH returns the arguments of env that run gh with args, and GH_TELEMETRY=0, so that the
// commands that read and set the setting send no telemetry themselves.
func quietGH(args ...string) []string {
	return append([]string{"GH_TELEMETRY=0", "gh"}, args...)
}
