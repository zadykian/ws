package values

import (
	"encoding/json"
	"slices"
	"testing"

	"go.yaml.in/yaml/v3"

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

// TestFansJSONIsTodaysText holds the file's form: two spaces of indent, and the names of its keys
// level by level, which the daemon reads. A renamed JSON tag fails it.
func TestFansJSONIsTodaysText(t *testing.T) {
	text, err := DefaultHost().Fans.JSON()
	if err != nil {
		t.Fatal(err)
	}
	c, err := daemon.Parse([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	if want, err := json.MarshalIndent(c, "", "  "); err != nil || text != string(want) {
		t.Errorf("JSON = %q, want the indent of two spaces: %q, %v", text, want, err)
	}
	var top map[string]any
	if err := json.Unmarshal([]byte(text), &top); err != nil {
		t.Fatal(err)
	}
	// Each level's keys, over all its entries: the optional ones show in some entry of the defaults.
	union := func(entries ...map[string]any) []string {
		var keys []string
		for _, e := range entries {
			for k := range e {
				if !slices.Contains(keys, k) {
					keys = append(keys, k)
				}
			}
		}
		slices.Sort(keys)
		return keys
	}
	list := func(key string) []map[string]any {
		var out []map[string]any
		for _, e := range top[key].([]any) {
			out = append(out, e.(map[string]any))
		}
		return out
	}
	curves := list("curves")
	for level, c := range map[string]struct{ got, want []string }{
		"file":    {union(top), []string{"controls", "curves", "interval", "sensors", "step"}},
		"step":    {union(top["step"].(map[string]any)), []string{"down", "up"}},
		"sensor":  {union(list("sensors")...), []string{"average", "hwmon", "label", "max", "name", "nvidia", "seconds"}},
		"curve":   {union(curves...), []string{"hysteresis", "name", "points", "response", "sensor"}},
		"control": {union(list("controls")...), []string{"curve", "hwmon", "label", "name", "percent"}},
	} {
		if !slices.Equal(c.got, c.want) {
			t.Errorf("%s keys = %q, want %q", level, c.got, c.want)
		}
	}
}

func TestFansJSONSurvivesTheValuesFile(t *testing.T) {
	f := DefaultHost().Fans
	want, err := f.JSON()
	if err != nil {
		t.Fatal(err)
	}
	data, err := yaml.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	var back Fans
	if err := yaml.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if got, err := back.JSON(); err != nil || got != want {
		t.Errorf("JSON after YAML = %q, %v; want %q", got, err, want)
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
