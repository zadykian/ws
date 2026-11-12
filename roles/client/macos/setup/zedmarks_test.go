package setup

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/mac"
)

func TestTheMacsZedFilesCarryTheMarksOfOwnership(t *testing.T) {
	first, _, _ := strings.Cut(string(mac.ZedSettings), "\n")
	if first != zedHeader {
		t.Errorf("global_settings.json starts with %q; want %q", first, zedHeader)
	}
	if !strings.HasPrefix(string(mac.ZedKeymap), zedKeymapMark) {
		t.Errorf("keymap.json does not start with %q", zedKeymapMark)
	}
}
