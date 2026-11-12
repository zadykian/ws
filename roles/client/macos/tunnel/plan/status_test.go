package tunnel

import (
	"testing"
	"time"
)

func TestFormat(t *testing.T) {
	now := time.Unix(1_761_900_100, 0)
	up := State{Tier: TierCable, Endpoint: "10.77.0.1:51820", Interface: "utun4",
		Since: time.Unix(1_761_900_000, 0), Routes: nets("10.1.0.0/16")}
	tests := []struct {
		name  string
		state State
		peer  Stats
		want  string
	}{
		{"up", up, Stats{Handshake: time.Unix(1_761_900_070, 0), Received: 1024, Sent: 2048, Known: true},
			"tier: cable\nendpoint: 10.77.0.1:51820\ninterface: utun4\nsince: 100 s ago\n" +
				"latest handshake: 30 s ago\ntransfer: 1024 B received, 2048 B sent\nroutes: 10.1.0.0/16\n"},
		{"left out", State{Tier: TierLAN, Endpoint: "192.168.1.10:51820", Interface: "utun4", Since: now,
			Routes: nets("10.1.0.0/16", "10.2.0.0/16"), LeftOut: nets("192.168.1.0/24")}, Stats{Known: true},
			"tier: lan\nendpoint: 192.168.1.10:51820\ninterface: utun4\nsince: 0 s ago\n" +
				"latest handshake: none\ntransfer: 0 B received, 0 B sent\nroutes: 10.1.0.0/16 10.2.0.0/16\n" +
				"left out, as they hold the endpoint: 192.168.1.0/24\n"},
		{"no path", State{Tier: TierNone, Interface: "utun4", Since: now}, Stats{},
			"tier: none, as no path to ws answers\nendpoint: none\ninterface: utun4\nsince: 0 s ago\n" +
				"latest handshake: none\nroutes: none\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.state.Format(now, tc.peer); got != tc.want {
				t.Errorf("got\n%s\nwant\n%s", got, tc.want)
			}
		})
	}
}
