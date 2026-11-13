package fans

import (
	"encoding/json"

	daemon "github.com/zadykian/ws/roles/server/ubuntu/fans/daemon"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// configPath is the file that ws server ubuntu fans reads.
const configPath = "/etc/ws-fans.json"

// pair converts a value for rising and one for falling.
func pair(p values.FanPair) daemon.Pair {
	return daemon.Pair{Up: p.Up, Down: p.Down}
}

// Config returns the text of /etc/ws-fans.json for the values. The daemon's own loader checks it
// before the role writes it, so that a config the daemon would refuse never replaces a good one.
func Config(f values.Fans) ([]byte, error) {
	c := daemon.Config{Interval: f.Interval, Step: pair(f.Step)}
	for _, s := range f.Sensors {
		c.Sensors = append(c.Sensors, daemon.SensorConfig{
			Name: s.Name, Hwmon: s.Hwmon, Label: s.Label, Nvidia: s.Nvidia,
			Average: s.Average, Seconds: s.Seconds, Max: s.Max,
		})
	}
	for _, cv := range f.Curves {
		c.Curves = append(c.Curves, daemon.CurveConfig{
			Name: cv.Name, Sensor: cv.Sensor, Points: cv.Points, Hysteresis: pair(cv.Hysteresis), Response: pair(cv.Response),
		})
	}
	for _, ct := range f.Controls {
		ctl := daemon.ControlConfig{Name: ct.Name, Hwmon: ct.Hwmon, Label: ct.Label, Curve: ct.Curve}
		ctl.Percent = ct.Percent
		c.Controls = append(c.Controls, ctl)
	}
	text, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return nil, err
	}
	text = append(text, '\n')
	if _, err := daemon.Parse(text); err != nil {
		return nil, err
	}
	return text, nil
}
