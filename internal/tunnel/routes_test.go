package tunnel

import (
	"net/netip"
	"reflect"
	"testing"
)

func nets(s ...string) []netip.Prefix {
	var out []netip.Prefix
	for _, each := range s {
		out = append(out, netip.MustParsePrefix(each))
	}
	return out
}

func TestParseRoutes(t *testing.T) {
	files := map[string][]byte{
		"b": []byte("172.16.0.0/12\n10.1.0.0/16"),
		"a": []byte("# lab\n\n 10.1.0.0/16 \n192.168.5.0/24\n"),
	}
	got, warnings := ParseRoutes(files)
	if want := nets("10.1.0.0/16", "192.168.5.0/24", "172.16.0.0/12"); !reflect.DeepEqual(got, want) {
		t.Errorf("networks %v, want %v", got, want)
	}
	if len(warnings) != 0 {
		t.Errorf("warnings %q", warnings)
	}
}

func TestParseRoutesLeavesOut(t *testing.T) {
	files := map[string][]byte{"lab": []byte("10.99.0.0/16\n10.99.0.5/32\n10.77.0.0/30\n0.0.0.0/0\nnope\n10.2.0.5/24\n")}
	got, warnings := ParseRoutes(files)
	// The first four overlap the tunnel's or the cable's subnet, the last two are no network.
	if len(got) != 0 {
		t.Errorf("networks %v, want none", got)
	}
	want := []string{
		"routes.d/lab: leaves out '10.99.0.0/16', not a network it can route",
		"routes.d/lab: leaves out '10.99.0.5/32', not a network it can route",
		"routes.d/lab: leaves out '10.77.0.0/30', not a network it can route",
		"routes.d/lab: leaves out '0.0.0.0/0', not a network it can route",
		"routes.d/lab: leaves out 'nope', not a network it can route",
		"routes.d/lab: leaves out '10.2.0.5/24', not a network it can route",
	}
	if !reflect.DeepEqual(warnings, want) {
		t.Errorf("warnings %q, want %q", warnings, want)
	}
}
