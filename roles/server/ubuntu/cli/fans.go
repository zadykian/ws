package cli

import (
	"context"
	"fmt"
	"io"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	fans "github.com/zadykian/ws/roles/server/ubuntu/fans/daemon"
)

// fansConfig is the config of ws server ubuntu fans, which the fans role writes.
const fansConfig = "/etc/ws-fans.json"

// newFansCmd returns ws server ubuntu fans: the board's fans on the curves of the config, until
// SIGTERM or SIGINT, then back on the board's own curves. Its log goes to stderr, the journal's.
func newFansCmd(p enginecli.Env) *cobra.Command {
	return &cobra.Command{
		Use:   "fans [CONFIG]",
		Short: "drive the board's fans on the curves of CONFIG until SIGTERM",
		Long:  "CONFIG is the file that the fans role writes, " + fansConfig + " by default.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			path := fansConfig
			if len(args) == 1 {
				path = args[0]
			}
			data, err := p.Machine.ReadFile(path)
			if err != nil {
				return err
			}
			cfg, err := fans.Parse(data)
			if err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
			return fansRun(c.Context(), cfg, c.ErrOrStderr())
		},
	}
}

// fansRun reads the GPUs by nvidia-smi and runs the daemon until a signal.
func fansRun(ctx context.Context, cfg fans.Config, stderr io.Writer) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	log := func(line string) { _, _ = fmt.Fprintln(stderr, line) }
	return fans.OnHost(ctx, cfg, log)
}
