// Package claude is the claude role as Go steps. They set up claude and cld, cld's completion in
// bash, claude's user settings and telemetry, its plugin marketplaces, and cld's restore at boot.
// See roles/server/ubuntu/claude/README.md.
package claude

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/systemd"
	"github.com/zadykian/ws/engine/core/step"
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
	a := account.Account{User: user}
	list := []step.Step{
		resource.Group{Title: installStep, Parts: []resource.Resource{Install{M: m, User: user}}},
		resource.Group{Title: updateStep, After: installStep, Parts: UpdateParts(v, a, m, u, out)},
		resource.Group{Title: "Claude cld completion", After: installStep, Parts: CompletionParts(v, a, m, u, out)},
		resource.Group{Title: "Claude user settings", After: updateStep, Parts: SettingsParts(v, a, m, u, out)},
		resource.Group{Title: "Claude telemetry", After: updateStep, Parts: TelemetryParts(v, a, m, u, out)},
	}
	if parts := MarketplacesParts(v, a, m, u, out); parts != nil {
		list = append(list, resource.Group{Title: "Claude marketplaces", After: "Claude telemetry", Parts: parts})
	}
	return append(list, user.WithAccount(m, "Claude restore", "Claude telemetry", func(a account.Account) ([]resource.Resource, error) {
		return append([]resource.Resource{systemd.Linger{M: m, User: user, Out: out}}, RestoreParts(v, a, m, u, out)...), nil
	}))
}
