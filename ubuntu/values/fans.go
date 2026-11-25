package values

import (
	"encoding/json"

	daemon "github.com/zadykian/ws/ubuntu/roles/fans/daemon"
)

// Fans is roles/fans' keys.
type Fans struct {
	// BoardVendor is the board's DMI vendor, which the driver is built for.
	BoardVendor string       `yaml:"WS_FANS_BOARD_VENDOR"`
	Sensors     []FanSensor  `yaml:"WS_FANS_SENSORS"`
	Curves      []FanCurve   `yaml:"WS_FANS_CURVES"`
	Controls    []FanControl `yaml:"WS_FANS_CONTROLS"`
	// Step is the most a fan's percent moves in a tick, up and down.
	Step FanPair `yaml:"WS_FANS_STEP"`
	// Interval is the seconds between two reads.
	Interval float64 `yaml:"WS_FANS_INTERVAL"`
}

// FanPair is a value for rising and a value for falling.
type FanPair struct {
	Up   float64 `yaml:"up"`
	Down float64 `yaml:"down"`
}

// FanSensor reads a temperature from one source: an hwmon label, an NVIDIA GPU, the average of
// another sensor over Seconds, or the highest of others.
type FanSensor struct {
	Name    string   `yaml:"name,omitempty"`
	Hwmon   string   `yaml:"hwmon,omitempty"`
	Label   string   `yaml:"label,omitempty"`
	Nvidia  *int     `yaml:"nvidia,omitempty"`
	Average string   `yaml:"average,omitempty"`
	Seconds float64  `yaml:"seconds,omitempty"`
	Max     []string `yaml:"max,omitempty"`
}

// FanCurve maps a sensor's temperature to a percent by its points, each [temperature, percent].
type FanCurve struct {
	Name       string       `yaml:"name,omitempty"`
	Sensor     string       `yaml:"sensor,omitempty"`
	Points     [][2]float64 `yaml:"points,omitempty"`
	Hysteresis FanPair      `yaml:"hysteresis,omitempty"`
	Response   FanPair      `yaml:"response,omitempty"`
}

// FanControl drives one fan by a curve, or holds it at Percent where that is set.
type FanControl struct {
	Name    string   `yaml:"name,omitempty"`
	Hwmon   string   `yaml:"hwmon,omitempty"`
	Label   string   `yaml:"label,omitempty"`
	Curve   string   `yaml:"curve,omitempty"`
	Percent *float64 `yaml:"percent,omitempty"`
}

// JSON returns the text of /etc/ws/fans.json, which ws ubuntu host fans reads, without its last
// newline. The daemon's own loader checks it first, so that a config the daemon would refuse fails
// the plan and never replaces a good one.
func (f Fans) JSON() (string, error) {
	c := daemon.Config{Interval: f.Interval, Step: daemonPair(f.Step)}
	for _, s := range f.Sensors {
		c.Sensors = append(c.Sensors, daemon.SensorConfig{
			Name: s.Name, Hwmon: s.Hwmon, Label: s.Label, Nvidia: s.Nvidia,
			Average: s.Average, Seconds: s.Seconds, Max: s.Max,
		})
	}
	for _, cv := range f.Curves {
		c.Curves = append(c.Curves, daemon.CurveConfig{
			Name: cv.Name, Sensor: cv.Sensor, Points: cv.Points, Hysteresis: daemonPair(cv.Hysteresis),
			Response: daemonPair(cv.Response),
		})
	}
	for _, ct := range f.Controls {
		c.Controls = append(c.Controls, daemon.ControlConfig{Name: ct.Name, Hwmon: ct.Hwmon, Label: ct.Label,
			Curve: ct.Curve, Percent: ct.Percent})
	}
	text, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	if _, err := daemon.Parse(text); err != nil {
		return "", err
	}
	return string(text), nil
}

// daemonPair is p as the daemon's config holds it.
func daemonPair(p FanPair) daemon.Pair { return daemon.Pair{Up: p.Up, Down: p.Down} }
