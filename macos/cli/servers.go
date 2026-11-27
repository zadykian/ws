package cli

import (
	"errors"
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/macos/values"
)

// newServersCmd returns ws macos servers.
func newServersCmd(p enginecli.Env, server *string) *cobra.Command {
	return &cobra.Command{
		Use:   "servers",
		Short: "list the Mac's servers with their target, slot and tunnel",
		Long: "It lists each server of servers/ beside values.yml: its target, its slot, and its tunnel, which is absent, " +
			"stopped, starting, unreadable, or the tier and endpoint it holds. A server whose file ws refuses shows - and its errors " +
			"follow the list, with exit code 1. A bad values.yml prints its refusal alone, with no row. It reads alone, as the user, and takes no --server.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			if *server != "" {
				return errors.New("servers lists every server: leave out --server")
			}
			mac := values.Path(p.Getenv, p.Pwd)
			fleet := values.ReadFleet(p.Machine, mac)
			// A bad values.yml is refused first and alone: no server is at fault.
			if fleet.MacErr != nil {
				return fleet.MacErr
			}
			if fleet.NamesErr != nil {
				return fleet.NamesErr
			}
			if len(fleet.Servers) == 0 {
				return fmt.Errorf("no server beside %s: write NAME/values.yml there, as macos/README.md says", mac)
			}
			var refused []error
			w := tabwriter.NewWriter(c.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tTARGET\tSLOT\tTUNNEL") //nolint:errcheck // tabwriter buffers, and Flush returns the error
			for _, s := range fleet.Servers {
				target, slot := "-", "-"
				if s.Err != nil {
					refused = append(refused, s.Err)
				} else {
					target, slot = s.Conf.Target, fmt.Sprint(s.Conf.TunnelSlot())
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, target, slot, tunnelOf(c.Context(), p.Machine, s.Name)) //nolint:errcheck // as above
			}
			return errors.Join(append(refused, w.Flush())...)
		},
	}
}

// pickServer returns the name of the server that --server names, or of the only one.
func pickServer(p enginecli.Env, server *string) (string, error) {
	return values.Pick(p.Machine, values.Path(p.Getenv, p.Pwd), *server)
}
