package daemon

import (
	"os"
	"strings"
	"testing"
)

const dtConfig = `{"interval": 1, "step": {"up": 100, "down": 100},
 "sensors": [{"name": "cpu", "hwmon": "coretemp", "label": "Package id 0"}],
 "curves": [{"name": "c", "sensor": "cpu", "points": [[30, 20], [70, 100]],
  "hysteresis": {"up": 0, "down": 0}, "response": {"up": 0, "down": 0}}],
 "controls": [{"name": "pump", "hwmon": "nct6687", "label": "Pump Fan", "curve": "c"}]}`

func TestTickSetsTheFanFromTheTemperatureAndLogsChangesOnce(t *testing.T) {
	root := driver(t)
	cfg, err := Parse([]byte(dtConfig))
	if err != nil {
		t.Fatal(err)
	}
	var logs []string
	d := New(cfg, Hwmon{OSSysfs{}, root}, nil, func(s string) { logs = append(logs, s) })
	for range 2 {
		if err := d.Tick(); err != nil {
			t.Fatal(err)
		}
	}
	// coretemp is not there: the fan goes back, and the problem is logged once for each.
	if len(logs) != 2 || !strings.HasPrefix(logs[0], "cpu: no temp labelled Package id 0") ||
		!strings.HasPrefix(logs[1], "pump: curve c has no temperature") {
		t.Fatalf("logs = %q", logs)
	}
}

func TestTickFollowsTheCurve(t *testing.T) {
	root := driver(t)
	core := root + "/hwmon0"
	writeAll(t, core, map[string]string{"name": "coretemp", "temp1_label": "Package id 0", "temp1_input": "50000"})
	cfg, _ := Parse([]byte(dtConfig))
	var logs []string
	d := New(cfg, Hwmon{OSSysfs{}, root}, nil, func(s string) { logs = append(logs, s) })
	if err := d.Tick(); err != nil {
		t.Fatal(err)
	}
	if got := file(t, root, "pwm2"); got != "153\n" { // 60 % of 255
		t.Errorf("pwm2 = %q, want 153", got)
	}
	if got := file(t, root, "pwm2_enable"); got != "1\n" {
		t.Errorf("pwm2_enable = %q, want 1", got)
	}
	d.Release()
	if got := file(t, root, "pwm2_enable"); got != "2\n" {
		t.Errorf("after Release pwm2_enable = %q, want 2", got)
	}
	if len(logs) != 0 {
		t.Errorf("logs = %q, want none", logs)
	}
}

func writeAll(t *testing.T, dir string, files map[string]string) {
	if err := os.Mkdir(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	for name, text := range files {
		if err := os.WriteFile(dir+"/"+name, []byte(text+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
