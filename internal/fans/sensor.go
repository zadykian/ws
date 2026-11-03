package fans

import (
	"strings"
	"time"
)

// Temperatures reads the NVIDIA GPUs' temperatures.
type Temperatures interface {
	Temperature(gpu int) (float64, error)
}

// reader reads one sensor's temperature at a time, given the values read so far in the tick.
type reader func(values map[string]*float64, now time.Time) (float64, error)

// sample is a reading of the sensor an average runs over.
type sample struct {
	at      time.Time
	celsius float64
}

// newReader returns the reader of a sensor that Check passed.
func newReader(s SensorConfig, hw Hwmon, gpus Temperatures) reader {
	switch {
	case s.Hwmon != "":
		return func(map[string]*float64, time.Time) (float64, error) { return hw.Temperature(s.Hwmon, s.Label) }
	case s.Nvidia != nil:
		return func(map[string]*float64, time.Time) (float64, error) { return gpus.Temperature(*s.Nvidia) }
	case s.Average != "":
		return average(s)
	}
	return func(values map[string]*float64, _ time.Time) (float64, error) {
		var gone []string
		highest := 0.0
		for i, name := range s.Max {
			if values[name] == nil {
				gone = append(gone, name)
			} else if i == 0 || *values[name] > highest {
				highest = *values[name]
			}
		}
		if len(gone) > 0 {
			return 0, missing("no reading of %s", strings.Join(gone, ", "))
		}
		return highest, nil
	}
}

// average returns the reader of the mean over the seconds of the sensor above.
func average(s SensorConfig) reader {
	var samples []sample
	window := time.Duration(s.Seconds * float64(time.Second))
	return func(values map[string]*float64, now time.Time) (float64, error) {
		if v := values[s.Average]; v != nil {
			samples = append(samples, sample{now, *v})
		}
		for len(samples) > 0 && !samples[0].at.After(now.Add(-window)) {
			samples = samples[1:]
		}
		if len(samples) == 0 {
			return 0, missing("no reading of %s in %v s", s.Average, s.Seconds)
		}
		sum := 0.0
		for _, x := range samples {
			sum += x.celsius
		}
		return sum / float64(len(samples)), nil
	}
}
