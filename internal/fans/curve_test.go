package fans

import "testing"

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
