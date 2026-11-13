package gpu

import (
	"context"
	"io"
	"os/exec"
)

// smiArgs make nvidia-smi print each GPU's index and temperature, a line each, every second.
var smiArgs = []string{"--query-gpu=index,temperature.gpu", "--format=csv,noheader,nounits", "-l", "1"}

// stream is the output of a child that ends with it: closing it stops the child.
type stream struct {
	io.ReadCloser
	cmd *exec.Cmd
}

// Close stops the child and waits for it.
func (s stream) Close() error {
	_ = s.ReadCloser.Close() // the pipe's read end; the child is next
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	_ = s.cmd.Wait() // the kill makes it fail
	return nil
}

// Command returns the start func of GPUs.Run for the command: its output is the stream. A missing
// nvidia-smi is the error of the start, and Run tries it again.
func Command(name string, args ...string) func(context.Context) (io.ReadCloser, error) {
	return func(ctx context.Context) (io.ReadCloser, error) {
		cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // the caller names the program: nvidia-smi, or a test's
		out, err := cmd.StdoutPipe()
		if err != nil {
			return nil, err
		}
		if err := cmd.Start(); err != nil {
			return nil, err
		}
		return stream{out, cmd}, nil
	}
}

// NvidiaSmi is the start func of GPUs.Run for the machine's nvidia-smi.
func NvidiaSmi() func(context.Context) (io.ReadCloser, error) {
	return Command("nvidia-smi", smiArgs...)
}
