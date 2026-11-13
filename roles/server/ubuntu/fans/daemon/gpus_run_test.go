package daemon

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/roles/server/ubuntu/fans/daemon/internal/missing"
)

func TestGPUsRunStartsTheStreamAgainWhereItEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	g := &GPUs{Retry: time.Millisecond}
	calls := 0
	start := func(context.Context) (io.ReadCloser, error) {
		calls++
		switch calls {
		case 1:
			return nil, errors.New("no nvidia-smi")
		case 2:
			return io.NopCloser(strings.NewReader("0, 45\n\n0, 46\n")), nil
		}
		// The third start follows the end of the second stream, which gave the latest reading.
		if _, err := g.Temperature(0); !missing.Is(err) || g.temps[0].celsius != 46 {
			t.Errorf("Temperature after the stream ended = %v, reading %v; want 46 but missing", err, g.temps[0])
		}
		cancel()
		return nil, errors.New("stop")
	}
	g.Run(ctx, start)
	if calls != 3 {
		t.Errorf("started the stream %d times, want 3", calls)
	}
}
