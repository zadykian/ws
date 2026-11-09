package bar

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestSmiArgs(t *testing.T) {
	if got := smiArgs(10 * time.Second); !slices.Equal(got[len(got)-2:], []string{"-l", "10"}) {
		t.Errorf("smiArgs(10 s) = %q, want -l 10", got)
	}
	if got := smiArgs(time.Millisecond); got[len(got)-1] != "1" {
		t.Errorf("smiArgs(1 ms) = %q, want -l 1", got)
	}
}

func TestGPUsReadTheHighestFreshReading(t *testing.T) {
	now := time.Unix(1000, 0)
	g := &gpus{now: func() time.Time { return now }}
	for _, line := range []string{"0, 3, 40", "1, 90, 35", "NVIDIA-SMI has failed", "2, [N/A], 50"} {
		g.record(line)
	}
	if use, c, ok := g.reading(20 * time.Second); !ok || use != 90 || c != 40 {
		t.Errorf("reading = %d %d %v, want 90 40 true", use, c, ok)
	}
	now = now.Add(30 * time.Second)
	if _, _, ok := g.reading(20 * time.Second); ok {
		t.Error("a reading older than its age counts")
	}
}

func TestGPUsRunStartsTheStreamAgain(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	g := &gpus{now: time.Now}
	starts := 0
	start := func(context.Context) (io.ReadCloser, error) {
		starts++
		switch starts {
		case 1:
			return nil, errors.New("no nvidia-smi")
		case 2:
			return io.NopCloser(strings.NewReader("0, 7, 45\n")), nil
		}
		cancel()
		return nil, ctx.Err()
	}
	g.run(ctx, start, time.Millisecond)
	if use, c, ok := g.reading(time.Minute); starts != 3 || !ok || use != 7 || c != 45 {
		t.Errorf("starts = %d, reading = %d %d %v; want 3 starts and 7 45", starts, use, c, ok)
	}
}
