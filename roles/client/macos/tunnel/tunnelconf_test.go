package tunnel

import (
	"testing"

	"github.com/zadykian/ws/roles/client/macos/conf"
)

func TestTunnelConf(t *testing.T) {
	head := "# Written by github.com/zadykian/ws: ws client macos apply, from ~/.config/ws/mac.conf.\nWS_PUBLIC_KEY=K\n"
	tests := []struct {
		name string
		c    conf.Conf
		want string
	}{
		{"default port", conf.Conf{PublicKey: "K"}, head + "WS_PORT=51820\nWS_SLOT=0\n"},
		{"all", conf.Conf{PublicKey: "K", LAN: "192.0.2.0/24", LANAddress: "192.0.2.5", Remote: "r.example", Port: "5", Slot: "3"},
			head + "WS_LAN=192.0.2.0/24\nWS_LAN_ADDRESS=192.0.2.5\nWS_REMOTE=r.example\nWS_PORT=5\nWS_SLOT=3\n"},
		{"the cable", conf.Conf{PublicKey: "K", CableService: "USB LAN"}, head + "WS_CABLE=yes\nWS_PORT=51820\nWS_SLOT=0\n"},
	}
	for _, tt := range tests {
		if got := tunnelConf(tt.c); got != tt.want {
			t.Errorf("%s: tunnelConf = %q; want %q", tt.name, got, tt.want)
		}
	}
}
