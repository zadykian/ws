package bar

import (
	"testing"
	"time"
)

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
