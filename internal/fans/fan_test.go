package fans

import (
	"os"
	"testing"
)

// driver makes the nct6687 device in a directory, with the Pump Fan as channel 2 at pwm 128 under
// the board's curve, and returns the directory.
func driver(t *testing.T) string {
	root := t.TempDir()
	dir := root + "/hwmon3"
	if err := os.Mkdir(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{"name": "nct6687", "fan1_label": "CPU Fan", "fan2_label": "Pump Fan", "pwm2": "128", "pwm2_enable": "2"} {
		if err := os.WriteFile(dir+"/"+name, []byte(text+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func file(t *testing.T, root, name string) string {
	b, err := os.ReadFile(root + "/hwmon3/" + name) //nolint:gosec // a file in the test's own directory
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestHwmonFanSpeedIsPwmInPercent(t *testing.T) {
	got, err := Hwmon{OSSysfs{}, driver(t)}.Fan("nct6687", "Pump Fan").Speed()
	if err != nil || got < 50.19 || got > 50.20 {
		t.Errorf("Speed = %v, %v; want 128/255 of 100", got, err)
	}
}

func TestHwmonFanIsMissingWhereTheHeaderIs(t *testing.T) {
	fan := Hwmon{OSSysfs{}, driver(t)}.Fan("nct6687", "Gone Fan")
	if _, err := fan.Speed(); !IsMissing(err) {
		t.Errorf("Speed = %v, want a missing error", err)
	}
}
