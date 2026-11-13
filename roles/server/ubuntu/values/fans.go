package values

// Fans is roles/fans' keys.
type Fans struct {
	// BoardVendor is the board's DMI vendor, which the driver is built for.
	BoardVendor string       `yaml:"fans_board_vendor"`
	Sensors     []FanSensor  `yaml:"fans_sensors"`
	Curves      []FanCurve   `yaml:"fans_curves"`
	Controls    []FanControl `yaml:"fans_controls"`
	// Step is the most a fan's percent moves in a tick, up and down.
	Step FanPair `yaml:"fans_step"`
	// Interval is the seconds between two reads.
	Interval float64 `yaml:"fans_interval"`
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
