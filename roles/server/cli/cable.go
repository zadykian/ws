package cli

import (
	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/roles/server/ubuntu/cable"
	"github.com/zadykian/ws/roles/server/values"
)

// newCableCmd returns ws server cable: the commands that the cable role's units run, on the
// service manager's units u.
func newCableCmd(p enginecli.Env, u effect.Units) *cobra.Command {
	return enginecli.Group("cable", "the cable role's check and reload, which its units run", newCableCheckCmd(p, u),
		newCableReloadCmd(p, u))
}

// newCableCheckCmd returns ws server cable check.
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

// newCableReloadCmd returns ws server cable reload.
func newCableReloadCmd(p enginecli.Env, u effect.Units) *cobra.Command {
	c := &cobra.Command{
		Use:   "reload",
		Short: "put netplan's files in force where the cable's file changed since the last reload",
		Long: "/var/lib/ws/cable-in-force holds the SHA-256 of the cable's file that the last reload put in force. Where " +
			"the file's differs, as where it is new, changed or gone, netplan generate writes networkd's files and " +
			"networkctl reload sets up each link whose file changed; then the mark takes the file's SHA-256, or goes " +
			"with the file. A failed reload leaves the mark, so the reload stays due. Another netplan file's change " +
			"never makes it due. Without booted systemd nothing is due. Before a reload it checks the live cable file " +
			"as cable check FILE does, so that another netplan file that takes the port never goes in force with " +
			"it.\n\n--check fails where a reload is due, and changes nothing. The cable role's run state runs it, " +
			"after the cable's file.",
		Args: cobra.NoArgs,
	}
	check := c.Flags().Bool("check", false, "fail where a reload is due, and change nothing")
	c.RunE = func(c *cobra.Command, _ []string) error {
		v, err := values.Load(p.Machine, serverVars)
		if err != nil {
			return err
		}
		term := effect.NewTerminal(c.ErrOrStderr())
		defer term.Flush()
		return settle(c.Context(), linesOf(term, c.CommandPath()),
			[]resource.Resource{cable.Reload{M: p.Machine, U: u, Cable: v.Cable}}, *check)
	}
	return c
}
