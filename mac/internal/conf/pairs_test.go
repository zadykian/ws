package conf

import (
	"strings"
	"testing"
)

func TestPairs(t *testing.T) {
	tests := []struct{ name, text, want string }{
		{"lan alone", "WS_LAN=10.88.0.0/24", "WS_LAN and WS_LAN_ADDRESS go together"},
		{"address alone", "WS_LAN_ADDRESS=10.88.0.2", "WS_LAN and WS_LAN_ADDRESS go together"},
		{"lan and address", "WS_LAN=10.88.0.0/24\nWS_LAN_ADDRESS=10.88.0.2", ""},
		{"network's own", "WS_LAN=10.88.0.0/24\nWS_LAN_ADDRESS=10.88.0.0", "WS_LAN_ADDRESS=10.88.0.0 is not a host's"},
		{"outside", "WS_LAN=10.88.0.0/24\nWS_LAN_ADDRESS=10.89.0.2", "inside WS_LAN, 10.88.0.0/24"},
		{"port alone", "WS_REMOTE_PORT=22", "WS_REMOTE_PORT needs WS_REMOTE"},
		{"cloud", "WS_TARGET=cloud", "WS_TARGET=cloud needs WS_REMOTE and WS_REMOTE_PORT"},
		{"cloud ok", "WS_TARGET=cloud\nWS_REMOTE=a.b\nWS_REMOTE_PORT=22", ""},
		{"mac-vm", "WS_TARGET=mac-vm", "WS_TARGET=mac-vm needs WS_LAN"},
		{"cloud with lan", "WS_TARGET=cloud\nWS_REMOTE=a.b\nWS_REMOTE_PORT=22\nWS_LAN=10.88.0.0/24\n" +
			"WS_LAN_ADDRESS=10.88.0.2\nWS_CABLE_PORT=en1", "has another target's key: WS_LAN WS_CABLE_PORT (docs"},
		{"mac-vm with remote", "WS_TARGET=mac-vm\nWS_LAN=10.88.0.0/24\nWS_LAN_ADDRESS=10.88.0.2\nWS_REMOTE=a.b",
			"has another target's key: WS_REMOTE ("},
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
