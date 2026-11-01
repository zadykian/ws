package sshcfg_test

import (
	"testing"

	"github.com/zadykian/ws/mac/internal/sshcfg"
)

func TestIncludeAt(t *testing.T) {
	const home = "/Users/me"
	tests := []struct {
		name  string
		text  string
		place sshcfg.Place
		line  int
	}{
		{"missing", "Host a\n", sshcfg.Missing, 0},
		{"first after comments", "# c\n\nInclude config.d/ws\nHost a\n", sshcfg.First, 3},
		{"quoted with the home", "Include \"/Users/me/.ssh/config.d/ws\" other\n", sshcfg.First, 1},
		{"tilde, any case, equals sign", "include = ~/.ssh/config.d/ws # c\n", sshcfg.First, 1},
		{"below a setting", "Include other\nInclude config.d/ws\n", sshcfg.Below, 2},
	}
	for _, tt := range tests {
		place, line := sshcfg.IncludeAt(tt.text, "config.d/ws", home)
		if place != tt.place || line != tt.line {
			t.Errorf("%s: IncludeAt = %v, %d; want %v, %d", tt.name, place, line, tt.place, tt.line)
		}
	}
}
