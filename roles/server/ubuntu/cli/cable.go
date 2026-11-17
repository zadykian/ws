package cli

import (
	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/server/ubuntu/cable"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// newCableCmd returns ws server ubuntu cable: the commands that the cable role's units run, on the
// service manager's units u.
func newCableCmd(p enginecli.Env, u effect.Units) *cobra.Command {
	return enginecli.Group("cable", "the cable role's checks, which its units run", newCableCheckCmd(p, u))
}

// newCableCheckCmd returns ws server ubuntu cable check.
func newCableCheckCmd(p enginecli.Env, u effect.Units) *cobra.Command {
	return &cobra.Command{
		Use:   "check [FILE]",
		Short: "refuse a cable's port that is an uplink, and a netplan FILE that would not set it up alone",
		Long: "It reads the values of " + serverVars + ". With no WS_CABLE_PORT there is no cable, and it passes. " +
			"It fails where the port is missing, carries a default route or holds a lease from DHCP, or where " +
			"networkd does not run on a booted machine.\n\nWith FILE, the cable's new netplan file, it also merges " +
			"FILE with netplan's other files in a scratch root, and fails where netplan refuses them or would hand " +
			"the port to another entry. It reads FILE as root, and no message quotes it. The cable role's guard " +
			"runs it, and its file's check with the new file.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			v, err := values.Load(p.Machine, serverVars)
			if err != nil {
				return err
			}
			file := ""
			if len(args) == 1 {
				file = args[0]
			}
			return cable.Check(c.Context(), p.Machine, u, v.Cable, file)
		},
	}
}
