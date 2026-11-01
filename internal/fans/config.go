// Package fans is ws-fans, the daemon of ws server fans: it drives the board's fans on the curves
// of the config in /etc/ws-fans.json, and hands them back to the board's own curves as it stops.
package fans

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
	Interval float64        `json:"interval"`
	Step     Pair           `json:"step"`
	Sensors  []SensorConfig `json:"sensors"`
}

// Pair is a value for rising and a value for falling.
type Pair struct {
	Up   float64 `json:"up"`
	Down float64 `json:"down"`
}

// SensorConfig reads one temperature, from one source: the hwmon device's channel labelled Label,
// the NVIDIA GPU of index Nvidia, the average over Seconds of the sensor named Average, or the
// highest of the sensors named in Max. A sensor names only sensors above it.
type SensorConfig struct {
	Name    string   `json:"name"`
	Hwmon   string   `json:"hwmon,omitempty"`
	Label   string   `json:"label,omitempty"`
	Nvidia  *int     `json:"nvidia,omitempty"`
	Average string   `json:"average,omitempty"`
	Seconds float64  `json:"seconds,omitempty"`
	Max     []string `json:"max,omitempty"`
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
