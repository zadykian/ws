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
	Interval float64 `json:"interval"`
	Step     Pair    `json:"step"`
}

// Pair is a value for rising and a value for falling.
type Pair struct {
	Up   float64 `json:"up"`
	Down float64 `json:"down"`
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
	return nil
}
