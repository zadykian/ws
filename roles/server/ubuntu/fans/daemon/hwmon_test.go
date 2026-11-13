package daemon

import (
	"os"
	"testing"

	"github.com/zadykian/ws/roles/server/ubuntu/fans/daemon/internal/missing"
)

// coretemp makes the CPU's hwmon device in a directory, and returns the directory.
func coretemp(t *testing.T) string {
	root := t.TempDir()
	dir := root + "/hwmon0"
	if err := os.Mkdir(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{"name": "coretemp", "temp1_label": "Package id 0", "temp1_input": "45500"} {
		if err := os.WriteFile(dir+"/"+name, []byte(text+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestTemperatureReadsTheLabelledChannel(t *testing.T) {
	got, err := Hwmon{OSSysfs{}, coretemp(t)}.Temperature("coretemp", "Package id 0")
	if err != nil || got != 45.5 {
		t.Errorf("Temperature = %v, %v; want 45.5", got, err)
	}
}

func TestTemperatureIsMissingWhereTheDeviceOrLabelIs(t *testing.T) {
	h := Hwmon{OSSysfs{}, coretemp(t)}
	for _, tc := range [][2]string{{"k10temp", "Tctl"}, {"coretemp", "Core 0"}} {
		_, err := h.Temperature(tc[0], tc[1])
		if !missing.Is(err) {
			t.Errorf("Temperature(%s) = %v, want a missing error", tc, err)
		}
	}
}
