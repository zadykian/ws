package main

import "github.com/spf13/cobra"

// plainFlag names the persistent flag that keeps a command to lines.
const plainFlag = "plain"

// drawsScreen tells whether a command draws a full-screen display: stdin and stderr are
// terminals, TERM names one that can draw, and --plain is not given. Anywhere else ws prints
// lines, which CI, pipes and the unit on ws read.
func drawsScreen(c *cobra.Command, p process) bool {
	plain, _ := c.Flags().GetBool(plainFlag)
	return screenWanted(plain, p.tty, p.errTTY, p.getenv("TERM"))
}

// screenWanted is the choice itself, on its four inputs.
func screenWanted(plain, stdinTTY, stderrTTY bool, term string) bool {
	return !plain && stdinTTY && stderrTTY && term != "" && term != "dumb"
}
