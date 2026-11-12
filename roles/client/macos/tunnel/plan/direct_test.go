package tunnel

import (
	"net/netip"
	"testing"
)

func TestDirectWay(t *testing.T) {
	lan := Conf{LAN: netip.MustParsePrefix("192.168.1.0/24"), LANServer: netip.MustParseAddr("192.168.1.10")}
	remote := lan
	remote.Remote = "ws.example.com"
	cable := Iface{"en5", netip.MustParseAddr("10.77.0.2")}
	home := Iface{"en0", netip.MustParseAddr("192.168.1.50")}
	away := Iface{"en0", netip.MustParseAddr("172.20.0.5")}
	tests := []struct {
		name   string
		ifaces []Iface
		conf   Conf
		port   string
		want   Direct
		found  bool
	}{
		{"the cable", []Iface{cable}, Conf{}, "", Direct{"10.77.0.1", "22"}, true},
		{"the LAN", []Iface{home}, remote, "22022", Direct{"192.168.1.10", "22"}, true},
		{"WS_REMOTE", []Iface{away}, remote, "22022", Direct{"ws.example.com", "22022"}, true},
		{"the cable before the LAN", []Iface{home, cable}, remote, "22022", Direct{"10.77.0.1", "22"}, true},
		{"the LAN before WS_REMOTE", []Iface{home}, remote, "22022", Direct{"192.168.1.10", "22"}, true},
		{"WS_REMOTE without its port", []Iface{away}, remote, "", Direct{}, false},
		{"WS_LAN without WS_LAN_ADDRESS", []Iface{home}, Conf{LAN: lan.LAN}, "", Direct{}, false},
		{"no LAN in the conf: the name and the port", []Iface{home},
			Conf{Remote: "ws.example.com"}, "22022", Direct{"ws.example.com", "22022"}, true},
		{"no address and no WS_REMOTE", nil, lan, "22022", Direct{}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, found := DirectWay(tc.ifaces, tc.conf, tc.port)
			if got != tc.want || found != tc.found {
				t.Errorf("got %v, %v; want %v, %v", got, found, tc.want, tc.found)
			}
		})
	}
}
