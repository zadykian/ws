package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac"
)

func TestEmbeddedFilesAreNotEmpty(t *testing.T) {
	for name, data := range map[string][]byte{
		"mac.Ghostty":     mac.Ghostty,
		"mac.TunnelPlist": mac.TunnelPlist,
		"mac.Zed":         mac.Zed,
		"mac.ZedKeymap":   mac.ZedKeymap,
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

func TestZedKeymapHasWsMarkAndIsJSONWithItsCommentsDropped(t *testing.T) {
	var lines []string
	for _, l := range strings.Split(string(mac.ZedKeymap), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(l), "//") {
			lines = append(lines, l)
		}
	}
	if !strings.HasPrefix(string(mac.ZedKeymap), "// Written by github.com/zadykian/ws") {
		t.Error("mac.ZedKeymap does not start with ws's mark")
	}
	if !json.Valid([]byte(strings.Join(lines, "\n"))) {
		t.Error("mac.ZedKeymap is not valid JSON under its comments")
	}
}
