package daemon

import (
	"context"
	"io"
	"os/exec"
)

// NetworkWatch is the command that prints a line each time configd posts a network change.
var NetworkWatch = []string{"notifyutil", "-w", "com.apple.system.config.network_change"}

// Child returns a start function for Watch: it runs the command and gives its standard output.
// Closing the reader ends the command and waits for it. It ends too where ctx ends.
func Child(argv ...string) func(context.Context) (io.ReadCloser, error) {
	return func(ctx context.Context) (io.ReadCloser, error) {
		cmd := exec.CommandContext(ctx, argv[0], argv[1:]...) //nolint:gosec // the caller names the command
		out, err := cmd.StdoutPipe()
		if err != nil {
			return nil, err
		}
		if err := cmd.Start(); err != nil {
			return nil, err
		}
		return &child{ReadCloser: out, cmd: cmd}, nil
	}
}

type child struct {
	io.ReadCloser
	cmd *exec.Cmd
}

func (c *child) Close() error {
	_ = c.cmd.Process.Kill()
	_ = c.cmd.Wait() // the kill makes it fail
	return nil
}

// Exit sends Exited when done closes, as wireguard-go's device does where it has stopped.
func (d *Daemon) Exit(ctx context.Context, done <-chan struct{}) {
	select {
	case <-ctx.Done():
	case <-done:
		d.Send(Exited)
	}
}
