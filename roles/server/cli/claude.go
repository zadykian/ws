package cli

import (
	"context"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/units"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/roles/server/ubuntu/claude"
	"github.com/zadykian/ws/roles/server/values"
)

// claudeReadme is the page that each of claude's commands points to.
const claudeReadme = "roles/server/ubuntu/claude/README.md"

// newClaudeCmd returns ws server claude: the procedures that the claude role's units run.
func newClaudeCmd(p enginecli.Env) *cobra.Command {
	return enginecli.Group("claude", "the claude role's procedures, which its units run",
		claudeProcedure(p, "update", "update cld where a newer release is out",
			"It reads cld's version and the release that cld's latest release leads to, over HTTP, and runs cld "+
				"update where that one is newer ("+claudeReadme+"#installs-and-updates).", claude.UpdateParts),
		claudeProcedure(p, "completion", "write cld's completion in bash where it differs from what cld prints",
			"cld completion bash prints the script, and cld setup completion bash writes it where the one in "+
				"place differs ("+claudeReadme+"#installs-and-updates).", claude.CompletionParts),
		claudeProcedure(p, "settings", "have cld add to claude's user settings what they lack",
			"cld setup config user adds the permission set of WS_CLAUDE_PERMISSIONS, cld's choices and the channel "+
				"of WS_CLAUDE_NOTIFICATIONS, and keeps each value already there. --check runs cld on a copy of the "+
				"file ("+claudeReadme+"#user-settings).", claude.SettingsParts),
		claudeProcedure(p, "telemetry", "set claude's telemetry keys in the env of its user settings",
			"It sets the keys of WS_CLAUDE_TELEMETRY_ENV and removes those that would send elsewhere, and names "+
				"each key, never its value ("+claudeReadme+"#telemetry).", claude.TelemetryParts),
		claudeProcedure(p, "marketplaces", "add the plugin marketplaces of WS_CLAUDE_MARKETPLACES that claude lacks",
			"An empty list asks claude nothing ("+claudeReadme+"#plugin-marketplaces).", claude.MarketplacesParts),
		claudeProcedure(p, "restore", "start WS_USER's systemd, and have it restore cld's sessions at boot",
			"cld setup restore writes the user unit. Without booted systemd it skips, and says so "+
				"("+claudeReadme+"#sessions-after-a-reboot).", claude.RestoreParts))
}

// claudeProcedure returns a command of claude's that settles the parts of proc, for WS_USER of the
// values of /etc/ws/values.yml. The role's run states run it as root, each after the one before.
func claudeProcedure(p enginecli.Env, use, short, long string, proc claude.Procedure) *cobra.Command {
	long += " The claude role's run state runs it as root."
	return procedure(use, short, long, func(ctx context.Context, out effect.Out) ([]resource.Resource, error) {
		v, err := values.Load(p.Machine, serverVars)
		if err != nil {
			return nil, err
		}
		a, err := account.User{Name: v.User, Home: v.Home}.Lookup(ctx, p.Machine)
		if err != nil {
			return nil, err
		}
		return proc(v, a, p.Machine, units.OSUnits{}, out), nil
	})
}
