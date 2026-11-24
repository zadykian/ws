package tunnel

import (
	"testing"

	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/macos/values"
)

func TestTunnelConf(t *testing.T) {
	head := file.Header("#") + "\nWS_PUBLIC_KEY=K\n"
	tests := []struct {
		name string
		c    values.Conf
		want string
	}{
		{"default port", values.Conf{PublicKey: "K"}, head + "WS_TUNNEL_PORT=51820\nWS_SLOT=0\n"},
		{"all", values.Conf{PublicKey: "K", LAN: "192.0.2.0/24", LANAddress: "192.0.2.5", Remote: "r.example", Port: "5", Slot: "3"},
			head + "WS_LAN=192.0.2.0/24\nWS_LAN_ADDRESS=192.0.2.5\nWS_REMOTE=r.example\nWS_TUNNEL_PORT=5\nWS_SLOT=3\n"},
		{"the cable", values.Conf{PublicKey: "K", CableService: "USB LAN"}, head + "WS_CABLE=yes\nWS_TUNNEL_PORT=51820\nWS_SLOT=0\n"},
		{"a guest's host's cable", values.Conf{PublicKey: "K", HostCable: true, Port: "51821"},
			head + "WS_CABLE=yes\nWS_TUNNEL_PORT=51821\nWS_SLOT=0\n"},
	}
	for _, tt := range tests {
		if got := tunnelConf(tt.c); got != tt.want {
			t.Errorf("%s: tunnelConf = %q; want %q", tt.name, got, tt.want)
		}
	}
}
