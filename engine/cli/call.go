package cli

import (
	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
)

// JSONFlag names the flag that prints a command's data.
const JSONFlag = "json"

// CallOf returns the call of command c in the process e: the facade reads the flags and the
// streams from it. A command without --json has no such flag, and the lookup then reads false.
func CallOf(c *cobra.Command, e Env) engine.Call {
	plain, _ := c.Flags().GetBool(PlainFlag)
	asJSON, _ := c.Flags().GetBool(JSONFlag)
	check, _ := c.Flags().GetBool("check")
	return engine.Call{
		Path: c.CommandPath(), In: c.InOrStdin(), Out: c.OutOrStdout(), Err: c.ErrOrStderr(),
		Plain: plain, JSON: asJSON, Check: check, Selected: c.Flags().Changed("only"),
		Machine: e.Machine, Getenv: e.Getenv,
		TTY: e.TTY, ErrTTY: e.ErrTTY, OutTTY: e.OutTTY, LockDir: e.LockDir(),
	}
}
