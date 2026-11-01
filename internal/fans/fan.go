package fans

import "strconv"

// pwmFull is the top of pwmN, the value of a fan at 100%.
const pwmFull = 255

// Fan is a fan header: its speed is read and set in %.
type Fan interface {
	// Speed returns the speed the fan is set to now.
	Speed() (float64, error)
	// Set sets the fan to the speed, unless it is set to it already.
	Set(percent float64) error
	// Release hands the fan back to the board's curve.
	Release() error
}

// HwmonFan is a header of a hwmon device, such as nct6687's, found by its fan's label.
type HwmonFan struct {
	h             Hwmon
	device, label string
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
