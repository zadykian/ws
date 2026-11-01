package fans

import (
	"strings"
	"testing"
)

// good is a config that holds every kind of value.
const good = `{
  "interval": 1,
  "step": {"up": 10, "down": 10}
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
	if c.Interval != 1 || c.Step != (Pair{10, 10}) {
		t.Errorf("config = %+v", c)
	}
}

func TestParseRefusesWhatIsNotAConfig(t *testing.T) {
	for name, tc := range map[string]struct{ old, replacement, want string }{
		"no JSON":        {"", "not json", "invalid"},
		"an unknown key": {`"interval": 1,`, `"interval": 1, "speed": 2,`, "unknown field"},
		"two values":     {"", good + good, "more than one"},
		"no interval":    {`"interval": 1`, `"interval": 0`, "interval"},
		"no step up":     {`"up": 10`, `"up": 0`, "step"},
	} {
		t.Run(name, func(t *testing.T) { refuses(t, tc.old, tc.replacement, tc.want) })
	}
}
