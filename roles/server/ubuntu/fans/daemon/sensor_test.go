package daemon

import (
	"testing"
	"time"
)

func celsius(v float64) *float64 { return &v }

func TestMaxIsTheHighestAndMissingWithAnyMissing(t *testing.T) {
	r := newReader(SensorConfig{Name: "m", Max: []string{"a", "b"}}, Hwmon{}, nil)
	now := time.Unix(0, 0)
	if got, err := r(map[string]*float64{"a": celsius(40), "b": celsius(55)}, now); err != nil || got != 55 {
		t.Errorf("max = %v, %v; want 55", got, err)
	}
	if got, err := r(map[string]*float64{"a": celsius(-5), "b": celsius(-9)}, now); err != nil || got != -5 {
		t.Errorf("max of negatives = %v, %v; want -5", got, err)
	}
	if _, err := r(map[string]*float64{"a": celsius(40), "b": nil}, now); !IsMissing(err) {
		t.Errorf("max with b missing = %v, want a missing error", err)
	}
}

func TestAverageIsTheMeanOverItsSeconds(t *testing.T) {
	r := newReader(SensorConfig{Name: "avg", Average: "a", Seconds: 10}, Hwmon{}, nil)
	start := time.Unix(100, 0)
	for _, c := range []struct {
		after time.Duration
		in    *float64
		want  float64
		gone  bool
	}{
		{0, celsius(40), 40, false},
		{5 * time.Second, celsius(50), 45, false},
		{10 * time.Second, celsius(60), 55, false}, // the first sample is 10 s old: out
		{30 * time.Second, nil, 0, true},
	} {
		got, err := r(map[string]*float64{"a": c.in}, start.Add(c.after))
		if c.gone != IsMissing(err) || (!c.gone && (err != nil || got != c.want)) {
			t.Errorf("at +%v: %v, %v; want %v, missing %v", c.after, got, err, c.want, c.gone)
		}
	}
}

type gpuFake map[int]float64

func (g gpuFake) Temperature(i int) (float64, error) {
	if v, ok := g[i]; ok {
		return v, nil
	}
	return 0, missing("no GPU %d", i)
}

func TestNvidiaSensorReadsTheGPU(t *testing.T) {
	one := 1
	r := newReader(SensorConfig{Name: "g", Nvidia: &one}, Hwmon{}, gpuFake{1: 61})
	if got, err := r(nil, time.Time{}); err != nil || got != 61 {
		t.Errorf("GPU 1 = %v, %v; want 61", got, err)
	}
}
