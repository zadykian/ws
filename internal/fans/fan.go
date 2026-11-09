package fans

import (
	"math"
	"strconv"
)

const (
	// pwmManual and pwmAuto are the values of pwmN_enable that hand the fan to pwmN, and back to
	// the board's own curve. pwmFull is pwmN's top.
	pwmManual = 1
	pwmAuto   = 2
	pwmFull   = 255
)

// Fan is a fan header: its speed is read and set in %.
type Fan interface {
	// Speed returns the speed the fan is set to now.
	Speed() (float64, error)
	// Set sets the fan to the speed, unless the fan runs at it already.
	Set(percent float64) error
	// Release hands the fan back to the board's curve.
	Release() error
}

// HwmonFan is a header of a hwmon device, such as nct6687's, found by its fan's label.
type HwmonFan struct {
	h             Hwmon
	device, label string
	written       int
	hasWritten    bool
}

// Fan returns the header of the device whose fan channel is labelled label. It touches nothing yet.
func (h Hwmon) Fan(device, label string) *HwmonFan {
	return &HwmonFan{h: h, device: device, label: label}
}

// Speed returns the speed the fan is set to now, in %.
func (f *HwmonFan) Speed() (float64, error) {
	path, err := f.h.file(f.device, "fan", f.label, "pwm{}")
	if err != nil {
		return 0, err
	}
	text, err := f.h.FS.Read(path)
	if err != nil {
		return 0, err
	}
	value, err := strconv.Atoi(text)
	if err != nil {
		return 0, missing("%s holds %q, not a number", path, text)
	}
	return float64(value) * percentFull / pwmFull, nil
}

// Set takes the fan from the board at the first speed, and writes the speed as pwmN's value.
func (f *HwmonFan) Set(percent float64) error {
	value := int(math.RoundToEven(percent * pwmFull / percentFull))
	if !f.hasWritten {
		enable, err := f.h.file(f.device, "fan", f.label, "pwm{}_enable")
		if err != nil {
			return err
		}
		if err := f.h.FS.Write(enable, strconv.Itoa(pwmManual)); err != nil {
			return err
		}
	}
	if f.hasWritten && value == f.written {
		return nil
	}
	pwm, err := f.h.file(f.device, "fan", f.label, "pwm{}")
	if err != nil {
		return err
	}
	if err := f.h.FS.Write(pwm, strconv.Itoa(value)); err != nil {
		return err
	}
	f.written, f.hasWritten = value, true
	return nil
}

// Release hands the fan back to the board's curve.
func (f *HwmonFan) Release() error {
	f.hasWritten = false
	enable, err := f.h.file(f.device, "fan", f.label, "pwm{}_enable")
	if err != nil {
		return err
	}
	return f.h.FS.Write(enable, strconv.Itoa(pwmAuto))
}
