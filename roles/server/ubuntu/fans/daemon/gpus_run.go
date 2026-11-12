package daemon

import (
	"bufio"
	"context"
	"io"
	"strings"
	"time"
)

// retry is how long Run waits before it starts the stream again.
const retry = 5 * time.Second

// fail notes why the stream gave nothing, which keeps every GPU missing until a reading comes.
func (g *GPUs) fail(why string) {
	g.mu.Lock()
	g.problem = why
	g.mu.Unlock()
}

// Run reads the stream of start until ctx ends. Where the stream ends or cannot start, Run starts
// it again after Retry: the driver may load after the daemon starts, or come back after an update.
func (g *GPUs) Run(ctx context.Context, start func(context.Context) (io.ReadCloser, error)) {
	wait := g.Retry
	if wait == 0 {
		wait = retry
	}
	for ctx.Err() == nil {
		g.stream(ctx, start)
		select {
		case <-ctx.Done():
		case <-time.After(wait):
		}
	}
}

// stream records the lines of one run of start.
func (g *GPUs) stream(ctx context.Context, start func(context.Context) (io.ReadCloser, error)) {
	r, err := start(ctx)
	if err != nil {
		g.fail(err.Error())
		return
	}
	defer func() { _ = r.Close() }() // a read only stream
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" {
			g.record(line)
		}
	}
	g.fail("nvidia-smi ended")
}
