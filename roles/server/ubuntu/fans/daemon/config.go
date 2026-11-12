// Package daemon is ws-fans, the daemon of ws server fans. It drives the board's fans on the curves
// of the config in /etc/ws-fans.json. As it stops, it hands them back to the board's own curves.
package daemon

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// percentFull is the top of a speed, in %.
const percentFull = 100

// Config is /etc/ws-fans.json. Interval is the tick in seconds and Step the most a fan's speed
// moves in a tick, in %.
type Config struct {
	Interval float64         `json:"interval"`
	Step     Pair            `json:"step"`
	Sensors  []SensorConfig  `json:"sensors"`
	Curves   []CurveConfig   `json:"curves"`
	Controls []ControlConfig `json:"controls"`
}

// Pair is a value for rising and a value for falling.
type Pair struct {
	Up   float64 `json:"up"`
	Down float64 `json:"down"`
}

// SensorConfig reads one temperature, from one source: the hwmon device's channel labelled Label,
// or the NVIDIA GPU of index Nvidia. The other sources are the average over Seconds of the sensor
// named Average, and the highest of the sensors named in Max. A sensor names only sensors above it.
type SensorConfig struct {
	Name    string   `json:"name"`
	Hwmon   string   `json:"hwmon,omitempty"`
	Label   string   `json:"label,omitempty"`
	Nvidia  *int     `json:"nvidia,omitempty"`
	Average string   `json:"average,omitempty"`
	Seconds float64  `json:"seconds,omitempty"`
	Max     []string `json:"max,omitempty"`
}

// CurveConfig is a graph of speed by temperature. Each point is [°C, %].
type CurveConfig struct {
	Name       string       `json:"name"`
	Sensor     string       `json:"sensor"`
	Points     [][2]float64 `json:"points"`
	Hysteresis Pair         `json:"hysteresis"`
	Response   Pair         `json:"response"`
}

// ControlConfig is a fan, the header of the hwmon device whose fan channel is labelled Label. It
// runs on the curve named Curve, or at Percent.
type ControlConfig struct {
	Name    string   `json:"name"`
	Hwmon   string   `json:"hwmon"`
	Label   string   `json:"label"`
	Curve   string   `json:"curve,omitempty"`
	Percent *float64 `json:"percent,omitempty"`
}

// Parse reads and checks a config. It refuses a key it does not know.
func Parse(data []byte) (Config, error) {
	var c Config
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return Config{}, err
	}
	if dec.More() {
		return Config{}, errors.New("more than one JSON value")
	}
	return c, c.Check()
}

// Check refuses a config that names what is not there, or holds a value the daemon cannot use.
func (c Config) Check() error {
	if c.Interval <= 0 {
		return fmt.Errorf("interval %v: not above 0", c.Interval)
	}
	if c.Step.Up <= 0 || c.Step.Down <= 0 {
		return errors.New("step: up and down must be above 0")
	}
	sensors := map[string]bool{}
	for _, s := range c.Sensors {
		if err := s.check(sensors); err != nil {
			return fmt.Errorf("sensor %s: %w", s.Name, err)
		}
		sensors[s.Name] = true
	}
	curves := map[string]bool{}
	for _, cv := range c.Curves {
		if err := cv.check(sensors); err != nil {
			return fmt.Errorf("curve %s: %w", cv.Name, err)
		}
		curves[cv.Name] = true
	}
	for _, ct := range c.Controls {
		if err := ct.check(curves); err != nil {
			return fmt.Errorf("fan %s: %w", ct.Name, err)
		}
	}
	return nil
}

// check refuses a sensor with no name, a name used twice (in above), no source or two, or a source
// that lacks what it needs.
func (s SensorConfig) check(above map[string]bool) error {
	if s.Name == "" || above[s.Name] {
		return errors.New("a name that is empty or used twice")
	}
	sources := 0
	for _, has := range []bool{s.Hwmon != "", s.Nvidia != nil, s.Average != "", len(s.Max) > 0} {
		if has {
			sources++
		}
	}
	if sources != 1 {
		return errors.New("one of hwmon, nvidia, average or max, and no more")
	}
	if s.Hwmon != "" && s.Label == "" {
		return errors.New("hwmon needs a label")
	}
	if s.Nvidia != nil && *s.Nvidia < 0 {
		return errors.New("nvidia: a GPU index below 0")
	}
	if s.Average != "" && (!above[s.Average] || s.Seconds <= 0) {
		return fmt.Errorf("average needs a sensor above it, not %q, and seconds above 0", s.Average)
	}
	for _, name := range s.Max {
		if !above[name] {
			return fmt.Errorf("no %s above it", name)
		}
	}
	return nil
}

// check refuses a curve with no name or no sensor of its own, points whose temperatures do not
// rise, or a speed outside 0-100%.
func (cv CurveConfig) check(sensors map[string]bool) error {
	if cv.Name == "" {
		return errors.New("no name")
	}
	if !sensors[cv.Sensor] {
		return fmt.Errorf("no sensor %s", cv.Sensor)
	}
	if len(cv.Points) == 0 {
		return errors.New("no points")
	}
	for i, p := range cv.Points {
		if i > 0 && p[0] <= cv.Points[i-1][0] {
			return errors.New("its points' temperatures do not rise")
		}
		if p[1] < 0 || p[1] > percentFull {
			return errors.New("a speed outside 0-100%")
		}
	}
	if cv.Hysteresis.Up < 0 || cv.Hysteresis.Down < 0 || cv.Response.Up < 0 || cv.Response.Down < 0 {
		return errors.New("a hysteresis or response time below 0")
	}
	return nil
}

// check refuses a fan with no header, or not exactly one of a curve that exists and a percent.
func (ct ControlConfig) check(curves map[string]bool) error {
	if ct.Name == "" || ct.Hwmon == "" || ct.Label == "" {
		return errors.New("a name, a hwmon device and a label")
	}
	if (ct.Curve == "") == (ct.Percent == nil) {
		return errors.New("a curve that exists, or a percent, and not both")
	}
	if ct.Curve != "" && !curves[ct.Curve] {
		return fmt.Errorf("no curve %s", ct.Curve)
	}
	if ct.Percent != nil && (*ct.Percent < 0 || *ct.Percent > percentFull) {
		return errors.New("a percent outside 0-100%")
	}
	return nil
}
