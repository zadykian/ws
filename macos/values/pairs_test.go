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
		{"ikeward set", Text("WS_IKEWARD_DNS", "10.20.0.53", "WS_IKEWARD_DOMAINS", "corp.test",
			"WS_IKEWARD_FLOORS", "10.20.0.0/16"), ""},
		{"ikeward dns alone", Text("WS_IKEWARD_DNS", "10.20.0.53"),
			"WS_IKEWARD_DNS, WS_IKEWARD_DOMAINS and WS_IKEWARD_FLOORS go together"},
		{"ikeward no floors", Text("WS_IKEWARD_DNS", "10.20.0.53", "WS_IKEWARD_DOMAINS", "corp.test"),
			"WS_IKEWARD_DNS, WS_IKEWARD_DOMAINS and WS_IKEWARD_FLOORS go together"},
		{"ikeward floors alone", Text("WS_IKEWARD_FLOORS", "10.20.0.0/16"),
			"WS_IKEWARD_DNS, WS_IKEWARD_DOMAINS and WS_IKEWARD_FLOORS go together"},
		{"ikeward dns outside", Text("WS_IKEWARD_DNS", "10.30.0.53", "WS_IKEWARD_DOMAINS", "corp.test",
			"WS_IKEWARD_FLOORS", "10.20.0.0/16 10.40.0.0/16"), "WS_IKEWARD_DNS=10.30.0.53 is in no floor of WS_IKEWARD_FLOORS"},
		{"ikeward domain resolved elsewhere", Text("WS_IKEWARD_DNS", "10.20.0.53", "WS_IKEWARD_DOMAINS",
			"corp.test", "WS_IKEWARD_FLOORS", "10.20.0.0/16", "WS_RESOLVERS", "corp.test=10.20.0.99"),
			"WS_IKEWARD_DOMAINS's corp.test is a WS_RESOLVERS domain too, which lacks WS_IKEWARD_DNS=10.20.0.53"},
		{"ikeward domain resolved alike", Text("WS_IKEWARD_DNS", "10.20.0.53", "WS_IKEWARD_DOMAINS",
			"corp.test", "WS_IKEWARD_FLOORS", "10.20.0.0/16", "WS_RESOLVERS",
			"corp.test=10.20.0.99,10.20.0.53 other.test=10.20.0.1"), ""},
		{"ikeward on a guest", Text("WS_HOST", "box", "WS_IKEWARD_DNS", "10.20.0.53"),
			"WS_IKEWARD_DNS, WS_IKEWARD_DOMAINS and WS_IKEWARD_FLOORS go together"},
		{"signing alone", Text("WS_GIT_SIGNING_PUB_KEY", "~/k.pub"), ""},
		{"agent alone", Text("WS_IDENTITY_AGENT", "~/a.sock"), "WS_IDENTITY_AGENT needs WS_IDENTITY_PUB_KEY"},
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
