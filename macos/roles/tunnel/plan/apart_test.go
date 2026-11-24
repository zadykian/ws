package plan

import (
	"net/netip"
	"reflect"
	"testing"
)

func TestApart(t *testing.T) {
	tests := []struct {
		name string
		own  []netip.Prefix
		held map[string][]netip.Prefix
		want []netip.Prefix
		logs []string
	}{
		{"none held", nets("10.1.0.0/16"), nil, nets("10.1.0.0/16"), nil},
		{"disjoint", nets("10.1.0.0/16"), map[string][]netip.Prefix{"lab": nets("10.2.0.0/16")},
			nets("10.1.0.0/16"), nil},
		{"equal", nets("10.1.0.0/16", "172.16.0.0/12"), map[string][]netip.Prefix{"lab": nets("10.1.0.0/16")},
			nets("172.16.0.0/12"), []string{"leaves 10.1.0.0/16 out: server lab's routes.d holds 10.1.0.0/16"}},
		{"own inside", nets("10.1.2.0/24"), map[string][]netip.Prefix{"lab": nets("10.1.0.0/16")}, nil,
			[]string{"leaves 10.1.2.0/24 out: server lab's routes.d holds 10.1.0.0/16"}},
		{"other inside", nets("10.1.0.0/16"), map[string][]netip.Prefix{"lab": nets("10.1.2.0/24")}, nil,
			[]string{"leaves 10.1.0.0/16 out: server lab's routes.d holds 10.1.2.0/24"}},
		{"two others", nets("10.1.0.0/16", "10.2.0.0/16", "10.3.0.0/16"),
			map[string][]netip.Prefix{"b": nets("10.2.0.0/16"), "a": nets("10.1.0.0/16")}, nets("10.3.0.0/16"),
			[]string{
				"leaves 10.1.0.0/16 out: server a's routes.d holds 10.1.0.0/16",
				"leaves 10.2.0.0/16 out: server b's routes.d holds 10.2.0.0/16",
			}},
		{"sorted servers", nets("10.1.0.0/16"),
			map[string][]netip.Prefix{"z": nets("10.1.0.0/16"), "a": nets("10.1.0.0/17")}, nil,
			[]string{"leaves 10.1.0.0/16 out: server a's routes.d holds 10.1.0.0/17"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, logs := Apart(tt.own, tt.held)
			if !reflect.DeepEqual(got, tt.want) || !reflect.DeepEqual(logs, tt.logs) {
				t.Errorf("Apart = %v, %q; want %v, %q", got, logs, tt.want, tt.logs)
			}
		})
	}
}
