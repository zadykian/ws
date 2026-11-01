package main

import (
	"testing"

	"github.com/zadykian/ws"
	"github.com/zadykian/ws/mac"
)

func TestEmbeddedFilesAreNotEmpty(t *testing.T) {
	for name, data := range map[string][]byte{
		"mac.Ghostty":      mac.Ghostty,
		"mac.TunnelPlist":  mac.TunnelPlist,
		"ws.ShellDefaults": ws.ShellDefaults,
	} {
		if len(data) == 0 {
			t.Errorf("%s is empty", name)
		}
	}
}
