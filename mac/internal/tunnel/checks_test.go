package tunnel

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
		{"ipv4", isIPv4, "192.168.1.5", true},
		{"ipv4 leading zero", isIPv4, "192.168.01.5", false},
		{"ipv4 three parts", isIPv4, "192.168.1", false},
		{"ipv4 out of range", isIPv4, "256.1.1.1", false},
		{"ipv4 v6", isIPv4, "::1", false},
		{"ipv4 text", isIPv4, "ws.example", false},
		{"name", isName, "ws.example.com", true},
		{"name one label", isName, "ws", true},
		{"name empty", isName, "", false},
		{"name trailing dot", isName, "ws.example.", false},
		{"name double dot", isName, "ws..example", false},
		{"name leading hyphen", isName, "-ws.example", false},
		{"name trailing hyphen", isName, "ws-.example", false},
		{"name underscore", isName, "w_s.example", false},
		{"name label of 63", isName, label63 + ".example", true},
		{"name label of 64", isName, label63 + "a.example", false},
		{"name of 254", isName, strings.Repeat(label63+".", 3) + strings.Repeat("a", 65), false},
		{"port", isPort, "51820", true},
		{"port 1", isPort, "1", true},
		{"port 65535", isPort, "65535", true},
		{"port 65536", isPort, "65536", false},
		{"port 0", isPort, "0", false},
		{"port leading zero", isPort, "080", false},
		{"port sign", isPort, "+80", false},
		{"port empty", isPort, "", false},
		{"key", isKey, key, true},
		{"key short", isKey, key[1:], false},
		{"key without =", isKey, strings.Repeat("A", 44), false},
		{"key with two =", isKey, strings.Repeat("A", 42) + "==", false},
		{"key with a hyphen", isKey, strings.Repeat("A", 42) + "-=", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.check(tc.in); got != tc.want {
				t.Errorf("%q: got %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseCIDR(t *testing.T) {
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
			if _, got := parseCIDR(tc.in); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
