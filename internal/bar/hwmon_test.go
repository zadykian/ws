package bar

import (
	"testing"
	"testing/fstest"
)

// sensors is a machine with two NVMe drives, a DIMM and Intel's cores.
func sensors() fstest.MapFS {
	file := func(s string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(s + "\n")} }
	return fstest.MapFS{
		"sys/class/hwmon/hwmon0/name":        file("nvme"),
		"sys/class/hwmon/hwmon0/temp1_label": file("Composite"),
		"sys/class/hwmon/hwmon0/temp1_input": file("41850"),
		"sys/class/hwmon/hwmon0/temp2_label": file("Sensor 1"),
		"sys/class/hwmon/hwmon0/temp2_input": file("60000"),
		"sys/class/hwmon/hwmon1/name":        file("nvme"),
		"sys/class/hwmon/hwmon1/temp1_label": file("Composite"),
		"sys/class/hwmon/hwmon1/temp1_input": file("44000"),
		"sys/class/hwmon/hwmon2/name":        file("spd5118"),
		"sys/class/hwmon/hwmon2/temp1_input": file("38250"),
		"sys/class/hwmon/hwmon3/name":        file("coretemp"),
		"sys/class/hwmon/hwmon3/temp1_label": file("Package id 0"),
		"sys/class/hwmon/hwmon3/temp1_input": file("52000"),
		"sys/class/hwmon/hwmon3/temp2_label": file("Core 0"),
		"sys/class/hwmon/hwmon3/temp2_input": file("70000"),
	}
}

func TestHottest(t *testing.T) {
	for _, c := range []struct {
		device, label string
		want          int
	}{
		{"nvme", "Composite", 44},
		{"spd5118", "", 38},
		{"coretemp", "Package id", 52},
	} {
		if got, ok := hottest(sensors(), c.device, c.label); !ok || got != c.want {
			t.Errorf("hottest(%s, %q) = %d, %v; want %d", c.device, c.label, got, ok, c.want)
		}
	}
	if _, ok := hottest(sensors(), "k10temp", ""); ok {
		t.Error("hottest read a device that the machine lacks")
	}
	broken := sensors()
	broken["sys/class/hwmon/hwmon2/temp1_input"] = &fstest.MapFile{Data: []byte("N/A\n")}
	if _, ok := hottest(broken, "spd5118", ""); ok {
		t.Error("hottest read a channel that holds no number")
	}
}
