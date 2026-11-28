package values

import (
	"strings"
	"testing"
)

func TestPairs(t *testing.T) {
	tests := []struct{ name, text, want string }{
		{"lan alone", Text("WS_LAN", "10.88.0.0/24"), "WS_LAN and WS_LAN_ADDRESS go together"},
		{"address alone", Text("WS_LAN_ADDRESS", "10.88.0.2"), "WS_LAN and WS_LAN_ADDRESS go together"},
		{"lan and address", Text("WS_LAN", "10.88.0.0/24", "WS_LAN_ADDRESS", "10.88.0.2"), ""},
		{"network's own", Text("WS_LAN", "10.88.0.0/24", "WS_LAN_ADDRESS", "10.88.0.0"), "WS_LAN_ADDRESS=10.88.0.0 is not a host's"},
		{"outside", Text("WS_LAN", "10.88.0.0/24", "WS_LAN_ADDRESS", "10.89.0.2"), "inside WS_LAN, 10.88.0.0/24"},
		{"port alone", Text("WS_REMOTE_PORT", "22"), "WS_REMOTE_PORT needs WS_REMOTE"},
		{"cloud", Text("WS_TARGET", "cloud"), "WS_TARGET=cloud needs WS_REMOTE and WS_REMOTE_PORT"},
		{"cloud ok", Text("WS_TARGET", "cloud", "WS_REMOTE", "a.b", "WS_REMOTE_PORT", "22"), ""},
		{"cloud with lan", Text(
			"WS_TARGET", "cloud",
			"WS_REMOTE", "a.b",
			"WS_REMOTE_PORT", "22",
			"WS_LAN", "10.88.0.0/24",
			"WS_LAN_ADDRESS", "10.88.0.2",
			"WS_CABLE_PORT", "en1",
		), "has another target's key: WS_LAN WS_CABLE_PORT (macos"},
	}
	for _, tt := range tests {
		_, errs := Parse(tt.text + "\n")
		switch {
		case tt.want == "" && len(errs) != 0:
			t.Errorf("%s: refused: %v", tt.name, errs)
		case tt.want != "" && (len(errs) != 1 || errs[0].Line != 0 || !strings.Contains(errs[0].Msg, tt.want)):
			t.Errorf("%s: %v, want %q", tt.name, errs, tt.want)
		}
	}
}
