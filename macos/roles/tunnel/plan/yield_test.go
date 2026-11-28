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

func TestYield(t *testing.T) {
	tests := []struct {
		name    string
		own     []string
		yielded []string
		want    []string
		logs    []string
	}{
		{"none yielded", []string{"10.20.0.0/16"}, nil, []string{"10.20.0.0/16"}, nil},
		{"equal", []string{"10.20.0.0/16", "172.16.0.0/12"}, []string{"10.20.0.0/16"}, []string{"172.16.0.0/12"},
			[]string{"leaves 10.20.0.0/16 out, as it yields 10.20.0.0/16 to another owner"}},
		{"own narrower", []string{"10.20.5.0/24"}, []string{"10.20.0.0/16"}, nil,
			[]string{"leaves 10.20.5.0/24 out, as it yields 10.20.0.0/16 to another owner"}},
		// The other owner's narrower routes win by length, so the wider network still takes the rest.
		{"own wider", []string{"10.0.0.0/8"}, []string{"10.20.0.0/16"}, []string{"10.0.0.0/8"}, nil},
		{"disjoint", []string{"10.21.0.0/16"}, []string{"10.20.0.0/16"}, []string{"10.21.0.0/16"}, nil},
		{"the narrowest names it", []string{"10.20.5.0/24"}, []string{"10.0.0.0/8", "10.20.0.0/16"}, nil,
			[]string{"leaves 10.20.5.0/24 out, as it yields 10.20.0.0/16 to another owner"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, logs := Yield(nets(tt.own...), nets(tt.yielded...))
			if !reflect.DeepEqual(got, nets(tt.want...)) || !reflect.DeepEqual(logs, tt.logs) {
				t.Errorf("Yield = %v, %q; want %v, %q", got, logs, tt.want, tt.logs)
			}
		})
	}
}

func TestHolders(t *testing.T) {
	got := Holders(
		nets("10.20.5.0/24", "10.20.6.0/24", "10.30.0.0/16", "10.40.0.0/16", "10.0.0.0/8"),
		nets("10.0.0.0/8", "10.20.0.0/16", "10.30.0.0/16", "10.99.0.0/16"))
	if want := nets("10.20.0.0/16", "10.30.0.0/16", "10.0.0.0/8"); !reflect.DeepEqual(got, want) {
		t.Errorf("Holders = %v, want %v: the narrowest holder of each, once", got, want)
	}
}

func TestWithRejects(t *testing.T) {
	tests := []struct {
		name                 string
		held, yield, rejects []string
		reject, unreject     []string
	}{
		{"none held", nil, nil, nil, nil, nil},
		{"held", []string{"10.20.0.0/16"}, []string{"10.20.0.0/16"}, nil, []string{"10.20.0.0/16"}, nil},
		{"held twice", []string{"10.20.0.0/16", "10.20.0.0/16"}, []string{"10.20.0.0/16"}, nil, []string{"10.20.0.0/16"}, nil},
		{"one it left", []string{"10.20.0.0/16"}, []string{"10.20.0.0/16"}, []string{"10.20.0.0/16"}, nil, nil},
		// The network stays out of its routes, so only the yield's prefix says where its reject goes.
		{"a yield that moved", []string{"10.0.0.0/8"}, []string{"10.0.0.0/8"}, []string{"10.20.0.0/16"},
			[]string{"10.0.0.0/8"}, []string{"10.20.0.0/16"}},
		{"one it left, no longer yielded", nil, []string{"10.30.0.0/16"}, []string{"10.20.0.0/16", "10.30.0.0/16"},
			nil, []string{"10.20.0.0/16"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoutePlan{}.WithRejects(nets(tt.held...), nets(tt.yield...), nets(tt.rejects...))
			if !reflect.DeepEqual(got.Reject, nets(tt.reject...)) || !reflect.DeepEqual(got.Unreject, nets(tt.unreject...)) {
				t.Errorf("Reject = %v, Unreject = %v; want %v, %v", got.Reject, got.Unreject, tt.reject, tt.unreject)
			}
		})
	}
}
