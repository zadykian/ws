package plan

import (
	"reflect"
	"testing"
)

func TestParseYield(t *testing.T) {
	got, warnings := ParseYield("10.20.0.0/16\n\n  10.30.0.0/16\nnope\n10.40.0.1/16\n10.20.0.0/16\nfd00::/8\n")
	if want := nets("10.20.0.0/16", "10.30.0.0/16"); !reflect.DeepEqual(got, want) {
		t.Errorf("ParseYield = %v, want %v", got, want)
	}
	want := []string{
		"the yield's line 'nope' is no IPv4 network; skips it",
		"the yield's line '10.40.0.1/16' is no IPv4 network; skips it",
		"the yield's line 'fd00::/8' is no IPv4 network; skips it",
	}
	if !reflect.DeepEqual(warnings, want) {
		t.Errorf("warnings = %q, want %q", warnings, want)
	}
	if got, warnings := ParseYield(""); got != nil || warnings != nil {
		t.Errorf("an empty yield = %v, %q; want none", got, warnings)
	}
}
