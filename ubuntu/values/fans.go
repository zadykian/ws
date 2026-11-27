package values

import (
	"encoding/json"

	daemon "github.com/zadykian/ws/ubuntu/roles/fans/daemon"
)

// Fans is roles/fans' keys.
type Fans struct {
	// BoardVendor is the board's DMI vendor, which the driver is built for.
	BoardVendor string                 `yaml:"WS_FANS_BOARD_VENDOR"`
	Sensors     []daemon.SensorConfig  `yaml:"WS_FANS_SENSORS"`
	Curves      []daemon.CurveConfig   `yaml:"WS_FANS_CURVES"`
	Controls    []daemon.ControlConfig `yaml:"WS_FANS_CONTROLS"`
	// Step is the most a fan's percent moves in a tick, up and down.
	Step daemon.Pair `yaml:"WS_FANS_STEP"`
	// Interval is the seconds between two reads.
	Interval float64 `yaml:"WS_FANS_INTERVAL"`
}

// JSON returns the text of /etc/ws/fans.json, which ws ubuntu host fans reads, without its last
// newline. The daemon's own check runs first, so that a config the daemon would refuse fails
// the plan and never replaces a good one.
func (f Fans) JSON() (string, error) {
	c := daemon.Config{Interval: f.Interval, Step: f.Step, Sensors: f.Sensors, Curves: f.Curves, Controls: f.Controls}
	if err := c.Check(); err != nil {
		return "", err
	}
	text, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", err
	}
	return string(text), nil
}
