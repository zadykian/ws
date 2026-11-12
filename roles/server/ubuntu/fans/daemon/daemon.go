package fans

import (
	"fmt"
	"time"
)

// Daemon drives the fans of a config on the curves, a tick at a time.
type Daemon struct {
	// Hwmon finds the devices' channels.
	Hwmon Hwmon
	// GPUs reads the NVIDIA GPUs' temperatures.
	GPUs Temperatures
	// Now returns the time. Nil means time.Now.
	Now func() time.Time
	// Log gets a line where a sensor or a fan has a problem, or is back, and at the start.
	Log func(string)

	cfg      Config
	names    []string
	readers  []reader
	curves   []*Curve
	controls []*control
	seen     map[string]string
}

// New returns the daemon of a config that Check passed.
func New(cfg Config, hw Hwmon, gpus Temperatures, log func(string)) *Daemon {
	d := &Daemon{Hwmon: hw, GPUs: gpus, Log: log, cfg: cfg, seen: map[string]string{}}
	byName := map[string]*Curve{}
	for _, s := range cfg.Sensors {
		d.names = append(d.names, s.Name)
		d.readers = append(d.readers, newReader(s, hw, gpus))
	}
	for _, cv := range cfg.Curves {
		c := NewCurve(cv)
		byName[cv.Name] = c
		d.curves = append(d.curves, c)
	}
	for _, ct := range cfg.Controls {
		d.controls = append(d.controls, &control{
			name: ct.Name, fan: hw.Fan(ct.Hwmon, ct.Label), curve: byName[ct.Curve],
			percent: ct.Percent, step: cfg.Step, held: true,
		})
	}
	return d
}

func (d *Daemon) now() time.Time {
	if d.Now != nil {
		return d.Now()
	}
	return time.Now()
}

// report logs the key's problem where it differs from the last one, "back" once it has gone.
func (d *Daemon) report(key, problem string) {
	if d.seen[key] == problem {
		return
	}
	d.seen[key] = problem
	if problem == "" {
		problem = "back"
	}
	d.Log(key + ": " + problem)
}

// Tick reads the sensors, moves the curves and steps the fans once. An error is a fan or a
// device that failed for a reason other than being missing.
func (d *Daemon) Tick() error {
	now, values := d.now(), map[string]*float64{}
	for i, read := range d.readers {
		name := d.names[i]
		t, err := read(values, now)
		switch {
		case err == nil:
			values[name] = &t
			d.report(name, "")
		case IsMissing(err):
			values[name] = nil
			d.report(name, err.Error())
		default:
			return fmt.Errorf("sensor %s: %w", name, err)
		}
	}
	for i, cv := range d.cfg.Curves {
		v := values[cv.Sensor]
		d.curves[i].Update(derefOr(v), v != nil, now)
	}
	for _, c := range d.controls {
		problem, err := c.tick()
		if err != nil {
			return fmt.Errorf("fan %s: %w", c.name, err)
		}
		d.report(c.name, problem)
	}
	return nil
}

func derefOr(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}
