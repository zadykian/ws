package check

import (
	"net/netip"
	"testing"
)

func TestLANHost(t *testing.T) {
	tests := []struct {
		lan, addr string
		want      bool
	}{
		{"10.88.0.0/24", "10.88.0.5", true},
		{"10.88.0.0/24", "10.88.0.1", true},
		{"10.88.0.0/24", "10.88.0.254", true},
		{"10.88.0.0/24", "10.88.0.0", false},
		{"10.88.0.0/24", "10.88.0.255", false},
		{"10.88.0.0/24", "10.88.1.5", false},
		{"10.88.0.0/30", "10.88.0.2", true},
		{"10.88.0.0/30", "10.88.0.3", false},
		{"10.88.0.0/31", "10.88.0.0", true},
		{"10.88.0.0/31", "10.88.0.1", true},
		{"10.88.0.7/32", "10.88.0.7", true},
		{"0.0.0.0/0", "0.0.0.0", false},
	}
	for _, tt := range tests {
		got := LANHost(netip.MustParsePrefix(tt.lan), netip.MustParseAddr(tt.addr))
		if got != tt.want {
			t.Errorf("LANHost(%s, %s) = %v, want %v", tt.lan, tt.addr, got, tt.want)
		}
	}
}

func TestNetworks(t *testing.T) {
	reserved := []netip.Prefix{netip.MustParsePrefix("10.77.0.0/30"), netip.MustParsePrefix("10.99.0.0/24")}
	tests := []struct {
		in   string
		want bool
	}{
		{"", true},
		{"10.20.0.0/16", true},
		{"10.20.0.0/16 \t172.16.0.0/12", true},
		{"10.20.0.0/16 10.99.0.0/25", false},
		{"10.77.0.0/16", false},
		{"0.0.0.0/0", false},
		{"10.20.0.1/16", false},
		{"10.20.0.0/16,10.30.0.0/16", false},
		{"*", false},
		{"10.20.0.0/16 ws", false},
	}
	for _, tt := range tests {
		if got := Networks(tt.in, reserved...); got != tt.want {
			t.Errorf("Networks(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
