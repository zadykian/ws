package cli

import (
	"time"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/roles/server/ubuntu/tunnel"
	"github.com/zadykian/ws/roles/server/values"
)

// newTunnelCmd returns ws server tunnel: the commands that the tunnel role's units run, on
// the service manager's units u.
func newTunnelCmd(p enginecli.Env, u effect.Units) *cobra.Command {
	return enginecli.Group("tunnel", "the tunnel role's check, wg0's key and wg0, which its units run",
		newTunnelCheckCmd(p, u), newTunnelKeyCmd(p), newTunnelWG0Cmd(p))
}

// newTunnelCheckCmd returns ws server tunnel check.
func newTunnelCheckCmd(p enginecli.Env, u effect.Units) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "refuse a networkd that does not run, and a peer's key file that holds no key",
		Long: "It reads the values of " + serverVars + ". It fails where networkd does not run on a booted machine, " +
			"as nothing would create wg0, or where a peer's /etc/wireguard/NAME.pub holds no WireGuard public key. " +
			"It names each peer whose file is missing, which wg0 leaves out, and prints the server's public key " +
			"where the key exists. The tunnel role's guard runs it, before wg0's key and files.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			v, err := values.Load(p.Machine, serverVars)
			if err != nil {
				return err
			}
			// The guard says each line of stdout.
			return tunnel.Check(c.Context(), p.Machine, u, effect.NewTerminal(c.OutOrStdout()), v.Tunnel)
		},
	}
}

// newTunnelKeyCmd returns ws server tunnel key.
func newTunnelKeyCmd(p enginecli.Env) *cobra.Command {
	c := &cobra.Command{
		Use:   "key",
		Short: "make wg0's private key where it is missing, and mend its owner, group and mode",
		Long: "The key is /etc/systemd/network/60-wg0.key, root:systemd-network 0640, which networkd reads. It is " +
			"made once, and never shown: the run that makes it prints its public key. A file that holds no private key fails.\n\n--check fails where the key " +
			"is to be made or mended, and changes nothing. The tunnel role's run state runs it, after the guard.",
		Args: cobra.NoArgs,
	}
	check := c.Flags().Bool("check", false, "fail where the key is to be made or mended, and change nothing")
	c.RunE = func(c *cobra.Command, _ []string) error {
		term := effect.NewTerminal(c.ErrOrStderr())
		defer term.Flush()
		out := linesOf(term, c.CommandPath())
		return settle(c.Context(), out, []resource.Resource{tunnel.Key{M: p.Machine, Out: out}}, *check)
	}
	return c
}

// newTunnelWG0Cmd returns ws server tunnel wg0.
func newTunnelWG0Cmd(p enginecli.Env) *cobra.Command {
	c := &cobra.Command{
		Use:   "wg0",
		Short: "make wg0 again where it lacks the key of its file, and wait for it to hold it",
		Long: "networkd reads wg0's private key only as it creates wg0. Where wg0 holds another key, it is deleted; " +
			"then networkctl reload creates it, and the command waits up to 10 s for it to hold the key, whose " +
			"public key it prints. Without booted systemd nothing is due.\n\n--check fails where wg0 is missing " +
			"or holds another key, and changes nothing. The tunnel role's run state runs it, after wg0's files.",
		Args: cobra.NoArgs,
	}
	check := c.Flags().Bool("check", false, "fail where wg0 lacks the key of its file, and change nothing")
	c.RunE = func(c *cobra.Command, _ []string) error {
		term := effect.NewTerminal(c.ErrOrStderr())
		defer term.Flush()
		lines := linesOf(term, c.CommandPath())
		w := tunnel.WG0{M: p.Machine, Out: lines, Tries: 10, Pause: time.Second}
		return settle(c.Context(), lines, []resource.Resource{w}, *check)
	}
	return c
}
