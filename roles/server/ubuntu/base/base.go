// Package base is the base role as Go steps: the packages every machine gets, the locale, the
// time zone, unattended upgrades and the swap file. See roles/server/ubuntu/base/README.md.
package base

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/apt"
	"github.com/zadykian/ws/engine/core/resource/locale"
	"github.com/zadykian/ws/engine/core/resource/timezone"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/ubuntu/base/internal/swap"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// The hints end a refusal of the locale or the time zone, which the values name.
const (
	localeHint   = "Set WS_BASE_LOCALE to one it lists, such as en_US.UTF-8"
	timezoneHint = "Set WS_BASE_TIMEZONE to another"
)

// Steps returns the role's steps in order: Packages, Locale, Timezone, Upgrades, Swap and
// Swappiness. The role takes no secrets.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, u effect.Units, out effect.Out) []step.Step {
	return []step.Step{
		resource.Group{Title: "Packages", Parts: []resource.Resource{apt.Packages{M: m, Out: out, Recommends: true, Names: v.Base.Packages}}},
		resource.Group{Title: "Locale", Parts: []resource.Resource{locale.Locale{M: m, Name: v.Base.Locale, Hint: localeHint}}},
		resource.Group{Title: "Timezone", Parts: []resource.Resource{timezone.Zone{M: m, Name: v.Base.Timezone, Hint: timezoneHint}}},
		resource.Group{Title: "Upgrades", Parts: upgradesParts(m, u, out)},
		resource.Group{Title: "Swap", Parts: swap.Parts(m, out, swap.Setting{SizeMiB: v.Base.SwapSizeMB, Swappiness: v.Base.Swappiness})},
		resource.Group{Title: "Swappiness", Parts: swap.Swappiness(m, out, v.Base.Swappiness)},
	}
}
