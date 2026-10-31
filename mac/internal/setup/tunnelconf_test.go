package setup

import (
	"testing"

	"github.com/zadykian/ws/mac/internal/conf"
)

func TestTunnelConf(t *testing.T) {
	head := "# Written by ws's mac/setup.sh from ~/.config/ws/mac.conf.\nWS_PUBLIC_KEY=K\n"
	tests := []struct {
		name string
		c    conf.Conf
		want string
	}{
		{"default port", conf.Conf{PublicKey: "K"}, head + "WS_PORT=51820\n"},
		{"all", conf.Conf{PublicKey: "K", LAN: "192.0.2.0/24", LANAddress: "192.0.2.5", Remote: "r.example", Port: "5"},
			head + "WS_LAN=192.0.2.0/24\nWS_LAN_ADDRESS=192.0.2.5\nWS_REMOTE=r.example\nWS_PORT=5\n"},
	}
	for _, tt := range tests {
		if got := tunnelConf(tt.c); got != tt.want {
			t.Errorf("%s: tunnelConf = %q; want %q", tt.name, got, tt.want)
		}
	}
}
