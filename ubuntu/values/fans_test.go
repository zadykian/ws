package values

import (
	"encoding/json"
	"testing"

	daemon "github.com/zadykian/ws/ubuntu/roles/fans/daemon"
)

func TestFansJSONOfTheDefaultsPassesTheDaemonsLoader(t *testing.T) {
	text, err := DefaultHost().Fans.JSON()
	if err != nil {
		t.Fatal(err)
	}
	c, err := daemon.Parse([]byte(text))
	if err != nil || len(c.Sensors) != 5 || len(c.Curves) != 3 || len(c.Controls) != 4 || c.Controls[3].Percent == nil {
		t.Errorf("Parse = %+v, %v", c, err)
	}
	if c.Curves[1].Points[1] != [2]float64{50, 20} || c.Sensors[1].Nvidia == nil || *c.Sensors[1].Nvidia != 0 {
		t.Errorf("curve gpu = %v, sensor gpu = %+v", c.Curves[1].Points, c.Sensors[1])
	}
}

// TestFansJSONIsTodaysText holds the file byte for byte: the config indented by two spaces, as the
// role wrote it before, with the last newline left to the template.
func TestFansJSONIsTodaysText(t *testing.T) {
	f := DefaultHost().Fans
	text, err := f.JSON()
	if err != nil {
		t.Fatal(err)
	}
	c, err := daemon.Parse([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	want, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if text != string(want) {
		t.Errorf("JSON = %q, want %q", text, want)
	}
}

func TestFansJSONKeepsFractionsOfAPercent(t *testing.T) {
	f := DefaultHost().Fans
	f.Curves[0].Hysteresis.Up = 1.5
	f.Step.Down = 0.5
	text, err := f.JSON()
	if err != nil {
		t.Fatal(err)
	}
	c, err := daemon.Parse([]byte(text))
	if err != nil || c.Curves[0].Hysteresis.Up != 1.5 || c.Step.Down != 0.5 {
		t.Errorf("Parse = %+v, %v; want 1.5 and 0.5 kept", c, err)
	}
}

func TestFansJSONRefusesWhatTheLoaderRefuses(t *testing.T) {
	for name, edit := range map[string]func(*Fans){
		"a curve with no sensor":   func(f *Fans) { f.Curves[0].Sensor = "none" },
		"a fan on a missing curve": func(f *Fans) { f.Controls[0].Curve = "none" },
		"no interval":              func(f *Fans) { f.Interval = 0 },
		"a sensor above itself":    func(f *Fans) { f.Sensors[4].Max = []string{"nothing"} },
	} {
		f := DefaultHost().Fans
		edit(&f)
		if _, err := f.JSON(); err == nil {
			t.Errorf("%s: JSON = nil error, want the loader's refusal", name)
		}
	}
}
