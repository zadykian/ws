package cli

import (
	"fmt"
	"net/netip"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/check"
	"github.com/zadykian/ws/macos/roles/tunnel"
)

// newTunnelYieldCmd returns ws macos tunnel yield.
func newTunnelYieldCmd(p enginecli.Env) *cobra.Command {
	return &cobra.Command{
		Use:   "yield [CIDR...]",
		Short: "yield the networks to another owner, or take them all back, as root",
		Args:  cobra.ArbitraryArgs,
		Long: "It writes " + tunnel.YieldFile + " and has each loaded daemon read it. A daemon stops routing each " +
			"of its networks that one of them holds, and leaves a -reject route there. With no network it removes " +
			"the file, and the daemons route their networks again. It returns once each daemon's state shows the " +
			"change, or fails after " + tunnel.YieldWait.String() + ".",
		RunE: func(c *cobra.Command, args []string) error {
			if p.Euid != 0 {
				return errNotRoot
			}
			var nets []netip.Prefix
			for _, a := range args {
				n, ok := check.CIDR(a)
				if !ok {
					return fmt.Errorf("'%s' is not an IPv4 network, as 192.0.2.0/24", a)
				}
				nets = append(nets, n)
			}
			return tunnel.Yielder{M: p.Machine}.Yield(c.Context(), nets)
		},
	}
}
