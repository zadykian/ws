package main

import (
	"encoding/json"
	"testing"

	"github.com/zadykian/ws/mac"
)

func TestEmbeddedFilesAreNotEmpty(t *testing.T) {
	for name, data := range map[string][]byte{
		"mac.Ghostty":     mac.Ghostty,
		"mac.TunnelPlist": mac.TunnelPlist,
		"mac.Zed":         mac.Zed,
	} {
		if len(data) == 0 {
			t.Errorf("%s is empty", name)
		}
	}
}

func TestZedSettingsAreJSON(t *testing.T) {
	if !json.Valid(mac.Zed) {
		t.Error("mac.Zed is not valid JSON")
	}
}
