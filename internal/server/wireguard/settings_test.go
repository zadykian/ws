package wireguard

import (
	"net/netip"
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/server/values"
)

func TestSettingsAddressAndPort(t *testing.T) {
	for name, tc := range map[string]struct {
		address string
		port    int
		want    string // a part of the refusal, or empty for none
	}{
		"the defaults":     {"10.99.0.1/24", 51820, ""},
		"another subnet":   {"192.168.7.1/24", 4000, ""},
		"port zero":        {"10.99.0.1/24", 0, "not a port from 1 to 65535"},
		"port too high":    {"10.99.0.1/24", 65536, "not a port from 1 to 65535"},
		"the last port":    {"10.99.0.1/24", 65535, ""},
		"no prefix":        {"10.99.0.1", 51820, "not an IPv4 address with its prefix length"},
		"not an address":   {"wg/24", 51820, "not an IPv4 address with its prefix length"},
		"IPv6":             {"fd00::1/64", 51820, "not an IPv4 address with its prefix length"},
		"the network":      {"10.99.0.0/24", 51820, "not a host's address in 10.99.0.0/24"},
		"the broadcast":    {"10.99.0.255/24", 51820, "not a host's address in 10.99.0.0/24"},
		"a point to point": {"10.99.0.0/31", 51820, ""},
		"a single address": {"10.99.0.1/32", 51820, ""},
	} {
		t.Run(name, func(t *testing.T) {
			err := Settings(values.Wireguard{Address: tc.address, Port: tc.port})
			switch {
			case tc.want == "" && err != nil:
				t.Errorf("Settings = %v, want none", err)
			case tc.want != "" && (err == nil || !strings.Contains(err.Error(), tc.want)):
				t.Errorf("Settings = %v, want a refusal with %q", err, tc.want)
			}
		})
	}
}

func TestSettingsNamesEveryProblem(t *testing.T) {
	err := Settings(values.Wireguard{Address: "10.99.0.255/24", Port: 0})
	if err == nil || !strings.Contains(err.Error(), "wireguard_port") || !strings.Contains(err.Error(), "wireguard_address") {
		t.Errorf("Settings = %v, want both the port and the address named", err)
	}
}

func TestLastIsTheBroadcast(t *testing.T) {
	for prefix, want := range map[string]string{"10.99.0.0/24": "10.99.0.255", "10.0.0.0/8": "10.255.255.255", "10.99.0.4/30": "10.99.0.7", "10.99.0.0/20": "10.99.15.255"} {
		if got := last(mustPrefix(t, prefix)).String(); got != want {
			t.Errorf("last(%s) = %s, want %s", prefix, got, want)
		}
	}
}

func mustPrefix(t *testing.T, s string) netip.Prefix {
	p, err := netip.ParsePrefix(s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
