package zed

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/resource/file"
)

func TestTheMacsZedFilesStartWithTheHeader(t *testing.T) {
	for name, text := range map[string][]byte{"global_settings.json": Settings, "keymap.json": Keymap} {
		first, _, _ := strings.Cut(string(text), "\n")
		if first != file.Header("//") {
			t.Errorf("%s starts with %q; want %q", name, first, file.Header("//"))
		}
	}
}
