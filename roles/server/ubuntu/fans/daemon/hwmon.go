package daemon

import (
	"path/filepath"
	"strconv"
	"strings"
)

const hwmonRoot = "/sys/class/hwmon"

// Hwmon finds the channels of hwmon devices by their labels.
type Hwmon struct {
	FS   Sysfs
	Root string // the directory of the devices, /sys/class/hwmon if empty
}

// file returns the file of attribute, with {} for the number of the channel of kind, temp or fan,
// labelled label on the device. It looks the channel up each time: hwmon numbers its devices anew
// when a driver loads again.
func (h Hwmon) file(device, kind, label, attribute string) (string, error) {
	root := h.Root
	if root == "" {
		root = hwmonRoot
	}
	for _, dir := range h.FS.Glob(root + "/hwmon*") {
		name, err := h.FS.Read(dir + "/name")
		if err != nil {
			return "", err
		}
		if name != device {
			continue
		}
		for _, lf := range h.FS.Glob(dir + "/" + kind + "*_label") {
			text, err := h.FS.Read(lf)
			if err != nil {
				return "", err
			}
			if text == label {
				number := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(lf), kind), "_label")
				return dir + "/" + strings.ReplaceAll(attribute, "{}", number), nil
			}
		}
	}
	return "", missing("no %s labelled %s on a hwmon device %s", kind, label, device)
}

// Temperature returns the temperature in °C of the channel labelled label.
func (h Hwmon) Temperature(device, label string) (float64, error) {
	path, err := h.file(device, "temp", label, "temp{}_input")
	if err != nil {
		return 0, err
	}
	text, err := h.FS.Read(path)
	if err != nil {
		return 0, err
	}
	milli, err := strconv.Atoi(text)
	if err != nil {
		return 0, missing("%s holds %q, not a number", path, text)
	}
	return float64(milli) / 1000, nil
}
