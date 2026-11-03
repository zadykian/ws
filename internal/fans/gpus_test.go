package fans

import (
	"strings"
	"testing"
	"time"
)

func TestGPUsTemperatureIsTheLatestReading(t *testing.T) {
	now := time.Unix(100, 0)
	g := &GPUs{Now: func() time.Time { return now }}
	g.record("0, 45")
	g.record("1, 50")
	g.record("0, 47")
	if got, err := g.Temperature(0); err != nil || got != 47 {
		t.Errorf("Temperature(0) = %v, %v; want 47", got, err)
	}
	now = now.Add(maxAge + time.Second)
	if _, err := g.Temperature(1); !IsMissing(err) {
		t.Errorf("Temperature(1) after %v = %v, want a missing error", maxAge, err)
	}
}

func TestGPUsTemperatureIsMissingWithoutAReadingOrWithABadLine(t *testing.T) {
	g := &GPUs{}
	if _, err := g.Temperature(0); !IsMissing(err) {
		t.Errorf("Temperature before a reading = %v, want a missing error", err)
	}
	for _, line := range []string{"45", "x, 45", "0, hot"} {
		g.record(line)
		if _, err := g.Temperature(0); !IsMissing(err) || !strings.Contains(err.Error(), line) {
			t.Errorf("Temperature after %q = %v, want a missing error that names it", line, err)
		}
	}
}
