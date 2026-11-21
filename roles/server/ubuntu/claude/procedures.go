// Package claude is the claude role's procedures, which no kind holds: cld's update, completion,
// user settings and telemetry, claude's plugin marketplaces, and cld's restore at boot. Commands of
// ws run them for the role's units. See roles/server/ubuntu/claude/README.md.
package claude

import (
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/systemd"
	"github.com/zadykian/ws/roles/server/ubuntu/claude/internal/cld"
	"github.com/zadykian/ws/roles/server/ubuntu/claude/internal/settings"
	"github.com/zadykian/ws/roles/server/values"
)

// A Procedure is a procedure of the role's that no kind holds. It gives the parts that a command of
// ws settles for a unit, from the values and root's account.
type Procedure func(v values.Values, a account.Account, m effect.Machine, u effect.Units, out effect.Out) []resource.Resource

// UpdateParts updates cld where a newer release is out.
func UpdateParts(_ values.Values, a account.Account, m effect.Machine, _ effect.Units, _ effect.Out) []resource.Resource {
	return []resource.Resource{cld.Update{M: m, User: a.User}}
}

// CompletionParts writes cld's completion in bash where it differs from what cld prints.
func CompletionParts(_ values.Values, a account.Account, m effect.Machine, _ effect.Units, _ effect.Out) []resource.Resource {
	return []resource.Resource{cld.Completion{M: m, User: a.User}}
}

// SettingsParts has cld add to claude's user settings what they lack. A cld older than setup
// config user has it once the update is done.
func SettingsParts(v values.Values, a account.Account, m effect.Machine, _ effect.Units, _ effect.Out) []resource.Resource {
	return []resource.Resource{
		settings.UserSettings{M: m, User: a.User, Permissions: v.Claude.Permissions, Notifications: v.Claude.Notifications},
	}
}

// TelemetryParts sets claude's telemetry keys in its user settings. The user's own write makes
// ~/.claude where claude's installer has not. An existing one keeps its mode: it holds claude's
// credentials.
func TelemetryParts(v values.Values, a account.Account, m effect.Machine, _ effect.Units, _ effect.Out) []resource.Resource {
	return []resource.Resource{settings.Telemetry{M: m, User: a.User, Env: v.Claude.TelemetryEnv}}
}

// MarketplacesParts adds the plugin marketplaces of WS_CLAUDE_MARKETPLACES. An empty list asks
// claude nothing, as its unit exists on every machine.
func MarketplacesParts(v values.Values, a account.Account, m effect.Machine, _ effect.Units, _ effect.Out) []resource.Resource {
	if len(v.Claude.Marketplaces) == 0 {
		return nil
	}
	return []resource.Resource{Marketplaces{M: m, User: a.User, List: v.Claude.Marketplaces}}
}

// RestoreParts starts the user's systemd, then has it restore cld's sessions as it starts. The
// unit's name holds the account's uid, which a spec cannot know, so it stays here. Lingering is
// the role's own unit, before.
func RestoreParts(_ values.Values, a account.Account, m effect.Machine, u effect.Units, out effect.Out) []resource.Resource {
	return []resource.Resource{
		systemd.Unit{M: m, U: u, Out: out, Name: fmt.Sprintf("user@%d.service", a.UID), Run: systemd.Running},
		cld.Restore{M: m, Account: a, Out: out},
	}
}
