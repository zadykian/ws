// Package claude is the claude role as Go steps. They set up claude and cld, cld's completion in
// bash, claude's user settings and telemetry, its plugin marketplaces, and cld's restore at boot.
// See docs/claude.md.
package claude

import (
	"fmt"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
)

const (
	installStep = "Claude install"
	updateStep  = "Claude cld update"
)

// Steps returns the role's steps in order. Each of cld's steps comes after the install, as cld is
// not there to ask until a run has installed it.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, u effect.Units, out effect.Out) []step.Step {
	user := host.User{Name: v.User, Home: v.Home}
	list := []step.Step{
		host.Group{Title: installStep, Parts: []host.Resource{Install{M: m, User: user}}},
		host.Group{Title: updateStep, After: installStep, Parts: []host.Resource{Update{M: m, User: user}}},
		host.Group{Title: "Claude cld completion", After: installStep, Parts: []host.Resource{
			Completion{M: m, User: user},
		}},
		// A cld older than setup config user has it once the update is done.
		host.Group{Title: "Claude user settings", After: updateStep, Parts: []host.Resource{
			UserSettings{M: m, User: user, Permissions: v.Claude.Permissions, Notifications: v.Claude.Notifications},
		}},
		// The user's own write makes ~/.claude where claude's installer has not, as a dry run on a new
		// machine runs no installer. An existing one keeps its mode: it holds claude's credentials.
		host.Group{Title: "Claude telemetry", After: updateStep, Parts: []host.Resource{
			Telemetry{M: m, User: user, Env: v.Claude.TelemetryEnv},
		}},
	}
	if len(v.Claude.Marketplaces) > 0 {
		list = append(list, host.Group{Title: "Claude marketplaces", After: "Claude telemetry", Parts: []host.Resource{
			Marketplaces{M: m, User: user, List: v.Claude.Marketplaces},
		}})
	}
	return append(list, user.WithAccount(m, "Claude restore", "Claude telemetry", func(a host.Account) ([]host.Resource, error) {
		return []host.Resource{
			host.Linger{M: m, User: user, Out: out},
			host.Unit{M: m, U: u, Out: out, Name: fmt.Sprintf("user@%d.service", a.UID), Run: host.Running},
			Restore{M: m, Account: a, Out: out},
		}, nil
	}))
}
