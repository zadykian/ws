package signoz

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/signoz/internal/api"
	"github.com/zadykian/ws/roles/server/ubuntu/signoz/internal/dashboards"
	"github.com/zadykian/ws/roles/server/ubuntu/signoz/internal/retention"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// Steps returns the role's steps, in the order they run. SigNoz keeps its data for the values'
// retention days and serves its API at their URL. The role takes no secret: the root user's
// password is drawn on the machine.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, _ effect.Units, out effect.Out) []step.Step {
	s := v.Signoz
	client := api.Client{URL: s.URL}
	return []step.Step{
		Files(m, s.Dir),
		Password(m, s.Dir),
		Up{M: m, Out: out, Dir: s.Dir},
		Ready{M: m, Out: out, API: client},
		retention.Retention{M: m, Out: out, API: client, Days: s.RetentionDays},
		dashboards.Dashboards{M: m, Out: out, API: client, Names: s.Dashboards},
	}
}
