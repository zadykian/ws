package tunnel

import (
	"net/netip"
	"reflect"
	"testing"
)

func TestCandidates(t *testing.T) {
	lan := Conf{LAN: netip.MustParsePrefix("192.168.1.0/24"), LANServer: netip.MustParseAddr("192.168.1.10")}
	remote := lan
	remote.Remote = "ws.example.com"
	cable := Iface{"en5", netip.MustParseAddr("10.77.0.2")}
	home := Iface{"en0", netip.MustParseAddr("192.168.1.50")}
	away := Iface{"en0", netip.MustParseAddr("172.20.0.5")}
	ws := netip.MustParseAddr("203.0.113.9")
	tests := []struct {
		name   string
		ifaces []Iface
		conf   Conf
		resolv netip.Addr
		want   []Candidate
		why    int
	}{
		{"cable, LAN and remote", []Iface{cable, home}, remote, ws, []Candidate{
			{TierCable, CableServer}, {TierLAN, lan.LANServer}, {TierRemote, ws}}, 0},
		{"LAN and remote", []Iface{home}, remote, ws, []Candidate{{TierLAN, lan.LANServer}, {TierRemote, ws}}, 0},
		{"away: remote alone", []Iface{away}, remote, ws, []Candidate{{TierRemote, ws}}, 0},
		{"the name has no IPv4 address", []Iface{away}, remote, netip.Addr{}, nil, 1},
		{"WS_LAN without WS_LAN_ADDRESS", []Iface{home}, Conf{LAN: lan.LAN}, netip.Addr{}, nil, 0},
		{"no WS_LAN", []Iface{home}, Conf{}, netip.Addr{}, nil, 0},
		{"no address at all", nil, remote, ws, []Candidate{{TierRemote, ws}}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, why := Candidates(tc.ifaces, tc.conf, tc.resolv)
			if !reflect.DeepEqual(got, tc.want) || len(why) != tc.why {
				t.Errorf("got %v, %q; want %v with %d reasons", got, why, tc.want, tc.why)
			}
		})
	}
}
