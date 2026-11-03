package tunnel

import (
	"net/netip"
	"reflect"
	"testing"
)

func addrs(s ...string) []netip.Addr {
	var out []netip.Addr
	for _, each := range s {
		out = append(out, netip.MustParseAddr(each))
	}
	return out
}

func TestAllowedFor(t *testing.T) {
	routes := nets("10.1.0.0/16", "192.168.5.0/24", "203.0.113.0/24")
	tests := []struct {
		name string
		ends []netip.Addr
		told []netip.Prefix
		want []netip.Prefix
		out  []netip.Prefix
		logs int
		ips  string
	}{
		{"no endpoint", nil, nil, routes, nil, 0,
			"10.99.0.1/32 10.1.0.0/16 192.168.5.0/24 203.0.113.0/24"},
		{"an endpoint inside a network", addrs("192.168.5.20"), nil,
			nets("10.1.0.0/16", "203.0.113.0/24"), nets("192.168.5.0/24"), 1,
			"10.99.0.1/32 10.1.0.0/16 203.0.113.0/24"},
		{"logged once", addrs("192.168.5.20"), nets("192.168.5.0/24"),
			nets("10.1.0.0/16", "203.0.113.0/24"), nets("192.168.5.0/24"), 0,
			"10.99.0.1/32 10.1.0.0/16 203.0.113.0/24"},
		{"two endpoints tried", addrs("10.9.9.9", "203.0.113.7"), nil,
			nets("10.1.0.0/16", "192.168.5.0/24"), nets("203.0.113.0/24"), 1,
			"10.99.0.1/32 10.1.0.0/16 192.168.5.0/24"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := AllowedFor(routes, tc.ends, tc.told)
			if !reflect.DeepEqual(got.Want, tc.want) || !reflect.DeepEqual(got.Out, tc.out) || len(got.Logs) != tc.logs {
				t.Errorf("got %+v, want %v out %v with %d logs", got, tc.want, tc.out, tc.logs)
			}
			ips := ""
			for i, p := range got.IPs() {
				if i > 0 {
					ips += " "
				}
				ips += p.String()
			}
			if ips != tc.ips {
				t.Errorf("AllowedIPs %q, want %q", ips, tc.ips)
			}
		})
	}
}

func TestAllowedForLog(t *testing.T) {
	got := AllowedFor(nets("192.168.5.0/24"), addrs("192.168.5.20"), nil)
	want := []string{"leaves 192.168.5.0/24 out of the tunnel while it holds the endpoint, 192.168.5.20"}
	if !reflect.DeepEqual(got.Logs, want) {
		t.Errorf("logs %q, want %q", got.Logs, want)
	}
}
