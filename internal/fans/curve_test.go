package fans

import (
	"testing"
	"time"
)

func testCurve() *Curve {
	return NewCurve(CurveConfig{
		Name: "c", Sensor: "s", Points: [][2]float64{{30, 20}, {50, 30}, {80, 60}},
		Hysteresis: Pair{2, 3}, Response: Pair{5, 10},
	})
}

func TestSpeedAt(t *testing.T) {
	c := testCurve()
	for temperature, want := range map[float64]float64{10: 20, 30: 20, 40: 25, 50: 30, 65: 45, 80: 60, 95: 60} {
		if got := c.SpeedAt(temperature); got != want {
			t.Errorf("SpeedAt(%v) = %v, want %v", temperature, got, want)
		}
	}
}

func TestUpdateHoldsWithinTheHysteresisAndTheResponseTime(t *testing.T) {
	c, start := testCurve(), time.Unix(0, 0)
	at := func(s int) time.Time { return start.Add(time.Duration(s) * time.Second) }
	steps := []struct {
		temperature float64
		second      int
		want        float64
		why         string
	}{
		{60, 0, 40, "the first reading sets the speed at once"},
		{61.9, 1, 40, "within the hysteresis up"},
		{62, 2, 40, "past the hysteresis, but not for the response time"},
		{62, 6, 40, "4 s of 5 s"},
		{62, 7, 42, "5 s past it: the speed follows"},
		{59.5, 8, 42, "within the hysteresis down"},
		{56, 9, 42, "past it down, for 10 s"},
		{56, 18, 42, "9 s of 10 s"},
		{56, 19, 36, "10 s: the speed follows"},
	}
	for _, s := range steps {
		c.Update(s.temperature, true, at(s.second))
		if got, ok := c.Speed(); !ok || got != s.want {
			t.Errorf("%s: speed = %v, %v; want %v", s.why, got, ok, s.want)
		}
	}
}

func TestUpdateFollowsAtOnceBeyondTheEndPoints(t *testing.T) {
	c, now := testCurve(), time.Unix(0, 0)
	c.Update(60, true, now)
	c.Update(85, true, now)
	if got, _ := c.Speed(); got != 60 {
		t.Errorf("speed beyond the last point = %v, want 60", got)
	}
}

func TestUpdateForgetsTheSpeedWithoutATemperature(t *testing.T) {
	c, now := testCurve(), time.Unix(0, 0)
	c.Update(60, true, now)
	c.Update(0, false, now)
	if _, ok := c.Speed(); ok {
		t.Error("a speed is known without a temperature")
	}
	c.Update(40, true, now)
	if got, ok := c.Speed(); !ok || got != 25 {
		t.Errorf("speed after the return = %v, %v; want 25", got, ok)
	}
}
