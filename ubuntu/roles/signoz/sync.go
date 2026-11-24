package signoz

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/ubuntu/roles/signoz/internal/api"
	"github.com/zadykian/ws/ubuntu/roles/signoz/internal/dashboards"
	"github.com/zadykian/ws/ubuntu/roles/signoz/internal/retention"
	"github.com/zadykian/ws/ubuntu/values"
)

// Sync returns the steps of ws ubuntu guest signoz sync, in the order they run. The wait for
// SigNoz's API comes first, as the retention and the dashboards both call it. SigNoz keeps its
// data for v's retention days and serves its API at v's URL.
func Sync(v values.Signoz, m effect.Machine, out effect.Out) []step.Step {
	client := api.Client{URL: v.URL}
	return []step.Step{
		Ready{API: client},
		retention.Retention{M: m, API: client, Days: v.RetentionDays},
		dashboards.Dashboards{Out: out, API: client, Names: v.Dashboards},
	}
}
