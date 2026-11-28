package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/macos/values"
)

// valuesChecked is the line of a pass. The frame's file holds it, and so does its fallback for a
// run with no ws yet, so the next plan finds the file unchanged.
const valuesChecked = "ws values: checked"

// newValuesCheckCmd returns ws macos values check.
func newValuesCheckCmd(p enginecli.Env, server *string) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "check each server's values.yml, and name each bad or missing key",
		Long: "It reads each server's file in servers/ of $WS_VALUES, else of /etc/ws. It exits 1 " +
			"with each bad or missing key and its file, and prints one line where all pass. It checks every server, and takes no --server.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			if *server != "" {
				return errors.New("values check checks every server: leave out --server")
			}
			if err := values.Check(p.Machine, values.Dir(p.Getenv, p.Pwd)); err != nil {
				return err
			}
			_, err := fmt.Fprintln(c.OutOrStdout(), valuesChecked)
			return err
		},
	}
}
