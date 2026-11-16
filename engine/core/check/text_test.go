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
		{"service", CableService("USB 10/100/1000 LAN"), true},
		{"service bare", CableService("Ethernet"), true},
		{"service disabled", CableService("*Ethernet"), false},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}
