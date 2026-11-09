package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/internal/bar"
	"github.com/zadykian/ws/internal/fans"
)

// barLine is the file of ws server bar, in the runtime directory of the tmux role's unit.
const barLine = "/run/ws-bar/line"

// newBarCmd returns ws server bar: the host's load, a line every interval, until SIGTERM or
// SIGINT. tmux's status lines read the file.
func newBarCmd(_ process) *cobra.Command {
	var interval time.Duration
	c := &cobra.Command{
		Use:   "bar [FILE]",
		Short: "write the host's load to FILE every interval until SIGTERM",
		Long:  "FILE is " + barLine + " by default, which tmux's status lines read.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			if interval < time.Second {
				return usageError{fmt.Errorf("--interval is %v, under 1s", interval)}
			}
			file := barLine
			if len(args) == 1 {
				file = args[0]
			}
			ctx, stop := signal.NotifyContext(c.Context(), syscall.SIGTERM, syscall.SIGINT)
			defer stop()
			b := &bar.Bar{FS: os.DirFS("/"), Interval: interval}
			return b.Run(ctx, file, fans.Command)
		},
	}
	c.Flags().DurationVar(&interval, "interval", 10*time.Second, "the time between two lines")
	return c
}
