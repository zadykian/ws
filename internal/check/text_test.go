package check

import (
	"strings"
	"testing"
)

func TestText(t *testing.T) {
	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{"port", CablePort("enp5s0"), true},
		{"port of 15", CablePort(strings.Repeat("a", 15)), true},
		{"port of 16", CablePort(strings.Repeat("a", 16)), false},
		{"port hyphen first", CablePort("-e"), false},
		{"service", CableService("USB 10/100/1000 LAN", true), true},
		{"service blank bare", CableService("USB LAN", false), false},
		{"service bare", CableService("Ethernet", false), true},
		{"service disabled", CableService("*Ethernet", true), false},
		{"name", GitName("Ada Lovelace", true), true},
		{"name blank bare", GitName("Ada Lovelace", false), false},
		{"name quote", GitName(`Ada "A"`, true), false},
		{"name backslash", GitName(`Ada\`, true), false},
		{"email", GitEmail("ws@example.com"), true},
		{"email plus", GitEmail("a.b+c%d@ex-ample.org"), true},
		{"email at first", GitEmail("@example.com"), false},
		{"email no at", GitEmail("example.com"), false},
		{"email two ats", GitEmail("a@b@example.com"), false},
		{"email bad host", GitEmail("a@-example.com"), false},
		{"email bad local", GitEmail("a b@example.com"), false},
		{"secret", HookSecret("ws-hook_1.x"), true},
		{"secret of 64", HookSecret(strings.Repeat("a", 64)), true},
		{"secret of 65", HookSecret(strings.Repeat("a", 65)), false},
		{"secret blank", HookSecret("a b"), false},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}
