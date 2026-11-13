// Package claude is the claude role as Go steps. They set up claude and cld, cld's completion in
// bash, claude's user settings and telemetry, its plugin marketplaces, and cld's restore at boot.
// See roles/server/ubuntu/claude/README.md.
package claude

import (
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/systemd"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/claude/internal/cld"
	"github.com/zadykian/ws/roles/server/ubuntu/claude/internal/settings"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

const (
	installStep = "Claude install"
	updateStep  = "Claude cld update"
)

// Steps returns the role's steps in order. Each of cld's steps comes after the install, as cld is
// not there to ask until a run has installed it.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, u effect.Units, out effect.Out) []step.Step {
	user := account.User{Name: v.User, Home: v.Home}
	list := []step.Step{
		resource.Group{Title: installStep, Parts: []resource.Resource{Install{M: m, User: user}}},
		resource.Group{Title: updateStep, After: installStep, Parts: []resource.Resource{cld.Update{M: m, User: user}}},
		resource.Group{Title: "Claude cld completion", After: installStep, Parts: []resource.Resource{
			cld.Completion{M: m, User: user},
		}},
		// A cld older than setup config user has it once the update is done.
		resource.Group{Title: "Claude user settings", After: updateStep, Parts: []resource.Resource{
			settings.UserSettings{M: m, User: user, Permissions: v.Claude.Permissions, Notifications: v.Claude.Notifications},
		}},
		// The user's own write makes ~/.claude where claude's installer has not, as a dry run on a new
		// machine runs no installer. An existing one keeps its mode: it holds claude's credentials.
		resource.Group{Title: "Claude telemetry", After: updateStep, Parts: []resource.Resource{
			settings.Telemetry{M: m, User: user, Env: v.Claude.TelemetryEnv},
		}},
	}
	if len(v.Claude.Marketplaces) > 0 {
		list = append(list, resource.Group{Title: "Claude marketplaces", After: "Claude telemetry", Parts: []resource.Resource{
			Marketplaces{M: m, User: user, List: v.Claude.Marketplaces},
		}})
	}
	return append(list, user.WithAccount(m, "Claude restore", "Claude telemetry", func(a account.Account) ([]resource.Resource, error) {
		return []resource.Resource{
			systemd.Linger{M: m, User: user, Out: out},
			systemd.Unit{M: m, U: u, Out: out, Name: fmt.Sprintf("user@%d.service", a.UID), Run: systemd.Running},
			cld.Restore{M: m, Account: a, Out: out},
		}, nil
	}))
}
