package wizard

import "testing"

func TestParseStamp(t *testing.T) {
	for text, want := range map[string]stamp{
		"-":                   {},
		"v1 0 on - -":         {Version: "v1", Exit: "0"},
		"- 1 off aa bb":       {Exit: "1", Off: true, Secrets: "aa", Vars: "bb"},
		"v1 0 on - - surplus": {},
	} {
		if got := parseStamp(text); got != want {
			t.Errorf("parseStamp(%q) = %+v; want %+v", text, got, want)
		}
	}
}
