package fans

import (
	"testing"

	daemon "github.com/zadykian/ws/internal/fans"
	"github.com/zadykian/ws/internal/server/values"
)

func TestConfigOfTheDefaultsPassesTheDaemonsLoader(t *testing.T) {
	text, err := Config(values.Default().Fans)
	if err != nil {
		t.Fatal(err)
	}
	c, err := daemon.Parse(text)
	if err != nil || len(c.Sensors) != 5 || len(c.Curves) != 3 || len(c.Controls) != 4 || c.Controls[3].Percent == nil {
		t.Errorf("Parse = %+v, %v", c, err)
	}
	if c.Curves[1].Points[1] != [2]float64{50, 20} || c.Sensors[1].Nvidia == nil || *c.Sensors[1].Nvidia != 0 {
		t.Errorf("curve gpu = %v, sensor gpu = %+v", c.Curves[1].Points, c.Sensors[1])
	}
}

func TestConfigKeepsFractionsOfAPercent(t *testing.T) {
	f := values.Default().Fans
	f.Curves[0].Hysteresis.Up = 1.5
	f.Step.Down = 0.5
	text, err := Config(f)
	if err != nil {
		t.Fatal(err)
	}
	c, err := daemon.Parse(text)
	if err != nil || c.Curves[0].Hysteresis.Up != 1.5 || c.Step.Down != 0.5 {
		t.Errorf("Parse = %+v, %v; want 1.5 and 0.5 kept", c, err)
	}
}

func TestConfigRefusesWhatTheLoaderRefuses(t *testing.T) {
	for name, edit := range map[string]func(*values.Fans){
		"a curve with no sensor":   func(f *values.Fans) { f.Curves[0].Sensor = "none" },
		"a fan on a missing curve": func(f *values.Fans) { f.Controls[0].Curve = "none" },
		"no interval":              func(f *values.Fans) { f.Interval = 0 },
		"a sensor above itself":    func(f *values.Fans) { f.Sensors[4].Max = []string{"nothing"} },
	} {
		f := values.Default().Fans
		edit(&f)
		if _, err := Config(f); err == nil {
			t.Errorf("%s: Config = nil, want the loader's refusal", name)
		}
	}
}
