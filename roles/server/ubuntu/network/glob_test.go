package network

import "testing"

func TestGlobMatches(t *testing.T) {
	tests := []struct {
		glob, name string
		want       bool
	}{
		{"enp5s0", "enp5s0", true},
		{"enp5s0", "enp5s1", false},
		{"enp*", "enp5s0", true},
		{"enp*", "eth0", false},
		{"*", "", true},
		{"enp?s0", "enp5s0", true},
		{"enp?s0", "enp55s0", false},
		{"enp[45]s0", "enp5s0", true},
		{"enp[45]s0", "enp6s0", false},
		{"enp[!45]s0", "enp6s0", true},
		{"enp[!45]s0", "enp4s0", false},
		{"enp[0-9]s0", "enp7s0", true},
		{"[]]", "]", true},
		{"[!]]", "a", true},
		{"en[p", "en[p", true},
		{"en[p", "enp", false},
		{"a.b", "axb", false},
		{"a.b", "a.b", true},
		{"[^a]", "^", true},
		{"[^a]", "b", false},
	}
	for _, tt := range tests {
		if got := globMatches(tt.glob, tt.name); got != tt.want {
			t.Errorf("globMatches(%q, %q) = %v, want %v", tt.glob, tt.name, got, tt.want)
		}
	}
}
