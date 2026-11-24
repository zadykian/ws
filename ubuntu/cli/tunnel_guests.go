package cli

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/ubuntu/roles/incus"
	"github.com/zadykian/ws/ubuntu/roles/tunnel"
	"github.com/zadykian/ws/ubuntu/values"
)

// guestTunnels reads the host's values and each guest's own, and returns the tunnels that the host
// ends. It refuses a slot or a port that two tunnels share.
func guestTunnels(p enginecli.Env) ([]values.GuestTunnel, error) {
	h, err := values.LoadHost(p.Machine, hostVars)
	if err != nil {
		return nil, err
	}
	return values.GuestTunnels(p.Machine, h.Tunnel, h.Incus.Guests)
}

// newTunnelGuestsCmd returns ws ubuntu host tunnel guests: the commands that end each guest's
// tunnel on the host, which the tunnel role's units run.
func newTunnelGuestsCmd(p enginecli.Env, u effect.Units) *cobra.Command {
	return enginecli.Group("guests", "end each guest's tunnel on the host",
		guestsCheck(p, u),
		guestsProcedure(p, "key", "make the key of each guest's interface where it is missing",
			"Each interface wgN, N the guest's slot, reads /etc/systemd/network/60-wgN.key, root:systemd-network 0640. "+
				"A run makes the missing keys, and prints each new public key.",
			func(out effect.Out, t []values.GuestTunnel) []resource.Resource {
				return tunnel.GuestKeys(p.Machine, out, t)
			}),
		guestsProcedure(p, "links", "write each guest's interface, and make networkd create it with its key",
			"It writes 60-wgN.netdev and 60-wgN.network for each guest's slot, N, reloads networkd where a file or an "+
				"interface is due, deleting an interface of another key first, and waits up to 10 s for each new one.",
			func(out effect.Out, t []values.GuestTunnel) []resource.Resource {
				return []resource.Resource{tunnel.Links{M: p.Machine, Out: out, Tunnels: t, Tries: 10, Pause: time.Second}}
			}),
		guestsProcedure(p, "forward", "send what arrives on each guest's interface to that guest",
			"A rule on wgN selects the table of slot N, whose one route leads to the guest's address on the bridge. "+
				"The unit tunnel-forward.service runs it at boot, as the rule and the route do not outlive a reboot.",
			func(_ effect.Out, t []values.GuestTunnel) []resource.Resource {
				return []resource.Resource{tunnel.Forward{M: p.Machine, Tunnels: t, Bridge: incus.BridgeName}}
			}))
}

// guestsCheck returns ws ubuntu host tunnel guests check: the guard of the guests' tunnels.
func guestsCheck(p enginecli.Env, u effect.Units) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "refuse a clash of tunnels, a networkd that does not run, and a peer's key file that holds no key",
		Long: "It reads each guest's own values at /etc/ws/guests/NAME/values.yml. It fails where two tunnels share a slot " +
			"or a port, or where networkd does not run, or where a peer's /etc/wireguard/wgN/NAME.pub holds no WireGuard " +
			"public key. It names each peer whose file is missing, which the interface leaves out, and prints the public " +
			"key of each interface that has one. The tunnel role's guard runs it.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			t, err := guestTunnels(p)
			if err != nil {
				return err
			}
			return tunnel.GuestCheck(c.Context(), p.Machine, u, effect.NewTerminal(c.OutOrStdout()), t)
		},
	}
}

// guestsProcedure returns a command that settles the resources that build gives for the guests'
// tunnels, with --check.
func guestsProcedure(p enginecli.Env, use, short, long string, build func(effect.Out, []values.GuestTunnel) []resource.Resource) *cobra.Command {
	return procedure(use, short, long, func(_ context.Context, out effect.Out) ([]resource.Resource, error) {
		t, err := guestTunnels(p)
		if err != nil {
			return nil, err
		}
		return build(out, t), nil
	})
}
