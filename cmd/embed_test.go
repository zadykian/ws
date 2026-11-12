package main

import (
	"testing"

	"github.com/zadykian/ws/roles/client/macos/mac"
)

func TestEmbeddedFilesAreNotEmpty(t *testing.T) {
	for name, data := range map[string][]byte{
		"mac.Ghostty":     mac.Ghostty,
		"mac.TunnelPlist": mac.TunnelPlist,
	} {
		if len(data) == 0 {
			t.Errorf("%s is empty", name)
		}
	}
}
