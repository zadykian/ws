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
			"follow the list, with exit code 1. It reads alone, as the user, and takes no --server.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			if *server != "" {
				return errors.New("servers lists every server: leave out --server")
			}
			mac := values.Path(p.Getenv, p.Pwd)
			names, err := values.Servers(p.Machine, mac)
			if err != nil {
				return err
			}
			if len(names) == 0 {
				return fmt.Errorf("no server beside %s: write NAME/values.yml there, as macos/README.md says", mac)
			}
			var refused []error
			w := tabwriter.NewWriter(c.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tTARGET\tSLOT\tTUNNEL") //nolint:errcheck // tabwriter buffers, and Flush returns the error
			for _, name := range names {
				target, slot := "-", "-"
				if s, err := values.Load(p.Machine, mac, values.ServerPath(mac, name)); err != nil {
					refused = appendNew(refused, err)
				} else {
					target, slot = s.Target, fmt.Sprint(s.TunnelSlot())
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", name, target, slot, tunnelOf(c.Context(), p.Machine, name)) //nolint:errcheck // as above
			}
			return errors.Join(append(refused, w.Flush())...)
		},
	}
}

// appendNew adds err to errs unless one says the same: a bad values.yml fails every server's load.
func appendNew(errs []error, err error) []error {
	for _, e := range errs {
		if e.Error() == err.Error() {
			return errs
		}
	}
	return append(errs, err)
}

// pickServer returns the name of the server that --server names, or of the only one.
func pickServer(p enginecli.Env, server *string) (string, error) {
	return values.Pick(p.Machine, values.Path(p.Getenv, p.Pwd), *server)
}
