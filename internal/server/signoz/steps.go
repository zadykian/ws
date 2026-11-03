package signoz

import (
	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// Steps returns the role's steps, in the order they run. SigNoz keeps its data for the values'
// retention days and serves its API at their URL. The role takes no secret: the root user's
// password is drawn on the machine.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, out effect.Out) []step.Step {
	s := v.Signoz
	api := API{URL: s.URL}
	return []step.Step{
		Files(m, s.Dir),
		Password(m, s.Dir),
		Up{M: m, Out: out, Dir: s.Dir},
		Ready{M: m, Out: out, API: api},
		Retention{M: m, Out: out, API: api, Days: s.RetentionDays},
		Dashboards{M: m, Out: out, API: api, Names: s.Dashboards},
	}
}
