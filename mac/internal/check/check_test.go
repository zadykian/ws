package check

import (
	"strings"
	"testing"
)

func TestChecks(t *testing.T) {
	key := strings.Repeat("A", 43) + "="
	label63 := strings.Repeat("a", 63)
	tests := []struct {
		name  string
		check func(string) bool
		in    string
		want  bool
	}{
		{"ipv4", IPv4, "192.168.1.5", true},
		{"ipv4 leading zero", IPv4, "192.168.01.5", false},
		{"ipv4 three parts", IPv4, "192.168.1", false},
		{"ipv4 out of range", IPv4, "256.1.1.1", false},
		{"ipv4 v6", IPv4, "::1", false},
		{"ipv4 text", IPv4, "ws.example", false},
		{"name", Name, "ws.example.com", true},
		{"name one label", Name, "ws", true},
		{"name empty", Name, "", false},
		{"name trailing dot", Name, "ws.example.", false},
		{"name double dot", Name, "ws..example", false},
		{"name leading hyphen", Name, "-ws.example", false},
		{"name trailing hyphen", Name, "ws-.example", false},
		{"name underscore", Name, "w_s.example", false},
		{"name label of 63", Name, label63 + ".example", true},
		{"name label of 64", Name, label63 + "a.example", false},
		{"name of 254", Name, strings.Repeat(label63+".", 3) + strings.Repeat("a", 65), false},
		{"port", Port, "51820", true},
		{"port 1", Port, "1", true},
		{"port 65535", Port, "65535", true},
		{"port 65536", Port, "65536", false},
		{"port 0", Port, "0", false},
		{"port leading zero", Port, "080", false},
		{"port sign", Port, "+80", false},
		{"port empty", Port, "", false},
		{"key", Key, key, true},
		{"key short", Key, key[1:], false},
		{"key without =", Key, strings.Repeat("A", 44), false},
		{"key with two =", Key, strings.Repeat("A", 42) + "==", false},
		{"key with a hyphen", Key, strings.Repeat("A", 42) + "-=", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.check(tc.in); got != tc.want {
				t.Errorf("%q: got %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestCIDR(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"192.168.1.0/24", true},
		{"10.0.0.0/8", true},
		{"0.0.0.0/0", true},
		{"192.168.1.5/32", true},
		{"192.168.1.5/24", false},
		{"192.168.1.0/33", false},
		{"192.168.1.0/08", false},
		{"192.168.1.0", false},
		{"fd00::/8", false},
		{"ws.example/24", false},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			if _, got := CIDR(tc.in); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
