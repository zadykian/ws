// Package base is the base role as Go steps: the packages every machine gets, the locale, the
// time zone, unattended upgrades and the swap file. See docs/base.md.
package base

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
)

// Steps returns the role's steps in order: Packages, Locale, Timezone, Upgrades, Swap and
// Swappiness. The role takes no secrets.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, u effect.Units, out effect.Out) []step.Step {
	return []step.Step{
		host.Group{Title: "Packages", Parts: []host.Resource{host.Packages{M: m, Out: out, Recommends: true, Names: v.Base.Packages}}},
		host.Group{Title: "Locale", Parts: localeParts(m, v.Base.Locale)},
		host.Group{Title: "Timezone", Parts: timezoneParts(m, v.Base.Timezone)},
		host.Group{Title: "Upgrades", Parts: upgradesParts(m, u, out)},
		host.Group{Title: "Swap", Parts: swapParts(m, out, osDisk{}, swapSetting{SizeMiB: v.Base.SwapSizeMB, Swappiness: v.Base.Swappiness})},
		host.Group{Title: "Swappiness", Parts: swappinessParts(m, out, v.Base.Swappiness)},
	}
}
