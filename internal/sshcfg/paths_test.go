package sshcfg_test

import (
	"testing"

	"github.com/zadykian/ws/internal/sshcfg"
)

func TestExpand(t *testing.T) {
	for path, want := range map[string]string{
		"~/.ssh/k.pub": "/Users/me/.ssh/k.pub",
		"/etc/ws/x":    "/etc/ws/x",
		"~":            "~",
		"a/~/b":        "a/~/b",
		"~other/k.pub": "~other/k.pub",
		"~/":           "/Users/me/",
	} {
		if got := sshcfg.Expand(path, "/Users/me"); got != want {
			t.Errorf("Expand(%q) = %q, want %q", path, got, want)
		}
	}
}
