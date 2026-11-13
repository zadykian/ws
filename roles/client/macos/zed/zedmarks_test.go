package zed

import (
	"strings"
	"testing"
)

func TestTheMacsZedFilesCarryTheMarksOfOwnership(t *testing.T) {
	first, _, _ := strings.Cut(string(Settings), "\n")
	if first != zedHeader {
		t.Errorf("global_settings.json starts with %q; want %q", first, zedHeader)
	}
	if !strings.HasPrefix(string(Keymap), zedKeymapMark) {
		t.Errorf("keymap.json does not start with %q", zedKeymapMark)
	}
}
