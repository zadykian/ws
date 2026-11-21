package signoz

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/signoz/internal/api"
	"github.com/zadykian/ws/roles/server/ubuntu/signoz/internal/dashboards"
	"github.com/zadykian/ws/roles/server/ubuntu/signoz/internal/retention"
	"github.com/zadykian/ws/roles/server/values"
)

// Sync returns the steps of ws server signoz sync, in the order they run. The wait for
// SigNoz's API comes first, as the retention and the dashboards both call it. SigNoz keeps its
// data for v's retention days and serves its API at v's URL.
func Sync(v values.Signoz, m effect.Machine, out effect.Out) []step.Step {
	client := api.Client{URL: v.URL}
	return []step.Step{
		Ready{M: m, Out: out, API: client},
		retention.Retention{M: m, Out: out, API: client, Days: v.RetentionDays},
		dashboards.Dashboards{M: m, Out: out, API: client, Names: v.Dashboards},
	}
}
