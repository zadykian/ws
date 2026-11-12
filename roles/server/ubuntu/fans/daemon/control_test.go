package fans

import (
	"errors"
	"testing"
	"time"
)

var errBoom = errors.New("boom")

// fakeFan records what the control does to it.
type fakeFan struct {
	speed    float64
	err      error
	sets     []float64
	releases int
}

func (f *fakeFan) Speed() (float64, error) { return f.speed, f.err }
func (f *fakeFan) Set(p float64) error {
	f.sets = append(f.sets, p)
	return f.err
}
func (f *fakeFan) Release() error { f.releases++; return nil }

func TestControlStepsTowardsItsPercent(t *testing.T) {
	fan := &fakeFan{speed: 30}
	full := 100.0
	c := &control{name: "c", fan: fan, percent: &full, step: Pair{Up: 25, Down: 5}}
	for range 3 {
		if p, err := c.tick(); p != "" || err != nil {
			t.Fatalf("tick = %q, %v", p, err)
		}
	}
	if want := []float64{55, 80, 100}; !equal(fan.sets, want) {
		t.Errorf("sets = %v, want %v", fan.sets, want)
	}
	low := 0.0
	c.percent = &low
	_, _ = c.tick()
	if got := fan.sets[3]; got != 95 {
		t.Errorf("step down set %v, want 95", got)
	}
}

func equal(a, b []float64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestControlWithoutATemperatureHandsTheFanBackOnce(t *testing.T) {
	fan := &fakeFan{speed: 30}
	c := &control{name: "c", fan: fan, curve: NewCurve(CurveConfig{Name: "k", Points: [][2]float64{{0, 0}, {1, 1}}}), step: Pair{Up: 1, Down: 1}, held: true}
	for range 2 {
		if p, err := c.tick(); p == "" || err != nil {
			t.Fatalf("tick = %q, %v; want a problem", p, err)
		}
	}
	if fan.releases != 1 {
		t.Errorf("released %d times a fan a killed daemon may have left; want 1", fan.releases)
	}
	c.curve.Update(0.5, true, time.Time{})
	_, _ = c.tick()
	c.curve.Update(0, false, time.Time{})
	_, _ = c.tick()
	_, _ = c.tick()
	if fan.releases != 2 {
		t.Errorf("released %d times, want 2", fan.releases)
	}
}

func TestControlReturnsWhatIsNotMissing(t *testing.T) {
	fan := &fakeFan{err: errBoom}
	zero := 0.0
	c := &control{name: "c", fan: fan, percent: &zero}
	if _, err := c.tick(); err == nil {
		t.Error("tick on a failing fan = nil, want its error")
	}
}
