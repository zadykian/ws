package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/zadykian/ws/internal/fans"
)

// fansConfig is the config of ws server fans, which the fans role writes.
const fansConfig = "/etc/ws-fans.json"

// fansCommand runs ws server fans [CONFIG]: the board's fans on the curves of the config, until
// SIGTERM or SIGINT, then back on the board's own curves. Its log goes to stderr, the journal's.
func fansCommand(ctx context.Context, args []string, stderr io.Writer) int {
	path := fansConfig
	if len(args) > 1 {
		_, _ = io.WriteString(stderr, serverUsage)
		return 2
	}
	for _, arg := range args {
		path = arg
	}
	data, err := os.ReadFile(path) //nolint:gosec // the path is the one argument of a command run by root
	if err == nil {
		var cfg fans.Config
		if cfg, err = fans.Parse(data); err == nil {
			return fansRun(ctx, cfg, stderr)
		}
	}
	_, _ = fmt.Fprintf(stderr, "ws server fans: %s: %v\n", path, err)
	return 1
}

// fansRun reads the GPUs by nvidia-smi and runs the daemon until a signal.
func fansRun(ctx context.Context, cfg fans.Config, stderr io.Writer) int {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	gpus := &fans.GPUs{}
	go gpus.Run(ctx, fans.NvidiaSmi())
	log := func(line string) { _, _ = fmt.Fprintln(stderr, line) }
	d := fans.New(cfg, fans.Hwmon{FS: fans.OSSysfs{}}, gpus, log)
	if err := d.Run(ctx); err != nil {
		log("ws server fans: " + err.Error())
		return 1
	}
	return 0
}
