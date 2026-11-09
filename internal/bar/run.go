package bar

import (
	"context"
	"io"
	"os"
	"time"
)

// smiRetry is how long the bar waits to start nvidia-smi again. Before the driver loads, or on a
// machine with none, it fails at once.
const smiRetry = time.Minute

// Command starts a program with its arguments and returns its output, which a close stops, as
// fans.Command does.
type Command func(name string, args ...string) func(context.Context) (io.ReadCloser, error)

// Run writes a line to file every Interval until ctx ends, then removes the file, so that tmux
// shows no load of a bar that has stopped. nvidia-smi runs by command for the whole run.
func (b *Bar) Run(ctx context.Context, file string, command Command) error {
	if b.gpus.now == nil {
		b.gpus.now = time.Now
	}
	ctx, stop := context.WithCancel(ctx)
	defer stop()
	go b.gpus.run(ctx, command("nvidia-smi", smiArgs(b.Interval)...), smiRetry)
	defer func() { _ = os.Remove(file) }() // a line never written leaves nothing to remove
	tick := time.NewTicker(b.Interval)
	defer tick.Stop()
	for {
		if err := writeLine(file, b.Line()); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
		}
	}
}

// writeLine replaces file with the line by a rename, so that tmux never reads half of one.
func writeLine(file, line string) error {
	next := file + ".new"
	if err := os.WriteFile(next, []byte(line+"\n"), 0o644); err != nil { //nolint:gosec // every user's tmux reads it
		return err
	}
	return os.Rename(next, file)
}
