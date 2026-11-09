package fans

import "time"

// Curve is a graph of speed by temperature, which holds still within its hysteresis. The speed
// follows the temperature once it has moved the hysteresis away from where the speed last changed,
// and stayed past it for the response time. Up and down each have their own. At or beyond the first
// or the last point, the speed follows at once.
type Curve struct {
	name   string
	sensor string
	points [][2]float64
	hyst   Pair
	resp   Pair

	temperature float64
	speed       float64
	known       bool // the speed is set from a temperature
	pending     bool
	way         bool // the pending move is up
	since       time.Time
}

// NewCurve returns the curve of a config that Check passed.
func NewCurve(c CurveConfig) *Curve {
	return &Curve{name: c.Name, sensor: c.Sensor, points: c.Points, hyst: c.Hysteresis, resp: c.Response}
}

// SpeedAt returns the speed at the temperature: flat before the first point and past the last.
func (c *Curve) SpeedAt(temperature float64) float64 {
	for i := 1; i < len(c.points); i++ {
		t0, s0, t1, s1 := c.points[i-1][0], c.points[i-1][1], c.points[i][0], c.points[i][1]
		if t0 <= temperature && temperature <= t1 {
			return s0 + (s1-s0)*(temperature-t0)/(t1-t0)
		}
	}
	if temperature < c.points[0][0] {
		return c.points[0][1]
	}
	return c.points[len(c.points)-1][1]
}

// Speed returns the speed in %, and false where the temperature is missing.
func (c *Curve) Speed() (float64, bool) { return c.speed, c.known }

// Update moves the speed to the temperature as the hysteresis allows. ok is false where the
// temperature is missing, which forgets the speed.
func (c *Curve) Update(temperature float64, ok bool, now time.Time) {
	if !ok {
		*c = Curve{name: c.name, sensor: c.sensor, points: c.points, hyst: c.hyst, resp: c.resp}
		return
	}
	last := len(c.points) - 1
	inside := c.points[0][0] < temperature && temperature < c.points[last][0]
	if c.known && inside {
		up := temperature > c.temperature
		hyst, resp := c.hyst.Down, c.resp.Down
		if up {
			hyst, resp = c.hyst.Up, c.resp.Up
		}
		diff := temperature - c.temperature
		if diff < 0 {
			diff = -diff
		}
		if diff < hyst {
			c.pending = false
			return
		}
		if !c.pending || c.way != up {
			c.pending, c.way, c.since = true, up, now
		}
		if now.Sub(c.since) < time.Duration(resp*float64(time.Second)) {
			return
		}
	}
	c.temperature, c.speed, c.known, c.pending = temperature, c.SpeedAt(temperature), true, false
}
