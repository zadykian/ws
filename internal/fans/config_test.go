package fans

import (
	"strings"
	"testing"
)

// good is a config that holds every kind of value.
const good = `{
  "interval": 1,
  "step": {"up": 10, "down": 10},
  "sensors": [
    {"name": "cpu", "hwmon": "coretemp", "label": "Package id 0"},
    {"name": "gpu", "nvidia": 0},
    {"name": "cpu_average", "average": "cpu", "seconds": 60},
    {"name": "both", "max": ["cpu_average", "gpu"]}
  ],
  "curves": [{"name": "c", "sensor": "both", "points": [[50, 30], [80, 60]],
    "hysteresis": {"up": 2, "down": 2}, "response": {"up": 5, "down": 5}}],
  "controls": [
    {"name": "Pump", "hwmon": "nct6687", "label": "Pump Fan", "curve": "c"},
    {"name": "MB", "hwmon": "nct6687", "label": "System Fan #2", "percent": 0}
  ]
}`

// refuses parses good with old replaced by replacement, or replacement alone where old is empty, and fails the
// test unless Parse refuses it with an error that holds want.
func refuses(t *testing.T, old, replacement, want string) {
	text := replacement
	if old != "" {
		text = strings.Replace(good, old, replacement, 1)
		if text == good {
			t.Fatalf("%q is not in the config", old)
		}
	}
	if _, err := Parse([]byte(text)); err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("Parse(%q) = %v, want an error with %q", text, err, want)
	}
}

func TestParseTakesTheGoodConfig(t *testing.T) {
	c, err := Parse([]byte(good))
	if err != nil {
		t.Fatal(err)
	}
	if c.Interval != 1 || c.Step != (Pair{10, 10}) || len(c.Sensors) != 4 || *c.Controls[1].Percent != 0 {
		t.Errorf("config = %+v", c)
	}
}

func TestParseRefusesWhatIsNotAConfig(t *testing.T) {
	for name, tc := range map[string]struct{ old, replacement, want string }{
		"no JSON":               {"", "not json", "invalid"},
		"an unknown key":        {`"interval": 1,`, `"interval": 1, "speed": 2,`, "unknown field"},
		"two values":            {"", good + good, "more than one"},
		"no interval":           {`"interval": 1`, `"interval": 0`, "interval"},
		"a sensor used twice":   {`"name": "gpu"`, `"name": "cpu"`, "used twice"},
		"two sources":           {`"nvidia": 0`, `"nvidia": 0, "hwmon": "x", "label": "y"`, "no more"},
		"no source":             {`"nvidia": 0`, `"label": "x"`, "one of"},
		"hwmon without a label": {`, "label": "Package id 0"`, ``, "needs a label"},
		"an average of a later": {`"average": "cpu"`, `"average": "both"`, "sensor above it"},
		"an average of no time": {`"seconds": 60`, `"seconds": 0`, "seconds"},
		"a max of a later one":  {`["cpu_average", "gpu"]`, `["cpu_average", "later"]`, "no later"},
		"a curve on no sensor":  {`"sensor": "both"`, `"sensor": "none"`, "no sensor none"},
		"points that fall":      {`[[50, 30], [80, 60]]`, `[[80, 30], [50, 60]]`, "do not rise"},
		"a speed past 100":      {`[80, 60]`, `[80, 101]`, "outside 0-100"},
		"a negative response":   {`"response": {"up": 5`, `"response": {"up": -5`, "below 0"},
		"a fan on no curve":     {`"curve": "c"`, `"curve": "d"`, "no curve d"},
		"a curve and a percent": {`"percent": 0`, `"percent": 0, "curve": "c"`, "not both"},
		"neither":               {`"percent": 0`, `"label": "x"`, "not both"},
		"a percent past 100":    {`"percent": 0`, `"percent": 120`, "percent"},
		"no step up":            {`"up": 10`, `"up": 0`, "step"},
	} {
		t.Run(name, func(t *testing.T) { refuses(t, tc.old, tc.replacement, tc.want) })
	}
}
