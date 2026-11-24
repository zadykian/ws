package sshcfg_test

import (
	"testing"

	"github.com/zadykian/ws/internal/sshcfg"
)

func TestIncludeAt(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		place sshcfg.Place
		line  int
	}{
		{"missing", "Host a\n", sshcfg.Missing, 0},
		{"first after comments", "# c\n\nInclude /etc/ws/ssh_config\nHost a\n", sshcfg.First, 3},
		{"quoted, with other paths after it", "Include \"/etc/ws/ssh_config\" other\n", sshcfg.First, 1},
		{"any case, equals sign, comment", "include = /etc/ws/ssh_config # c\n", sshcfg.First, 1},
		{"another file's Include", "Include ~/.ssh/config.d/ws\n", sshcfg.Missing, 0},
		{"below a setting", "Include other\nInclude /etc/ws/ssh_config\n", sshcfg.Below, 2},
	}
	for _, tt := range tests {
		place, line := sshcfg.IncludeAt(tt.text, sshcfg.File)
		if place != tt.place || line != tt.line {
			t.Errorf("%s: IncludeAt = %v, %d; want %v, %d", tt.name, place, line, tt.place, tt.line)
		}
	}
}
