package zed

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTheZedFilesAreJSONUnderTheirComments(t *testing.T) {
	for name, file := range map[string][]byte{"global_settings.json": Settings, "keymap.json": Keymap} {
		var lines []string
		for _, l := range strings.Split(string(file), "\n") {
			if !strings.HasPrefix(strings.TrimSpace(l), "//") {
				lines = append(lines, l)
			}
		}
		if !json.Valid([]byte(strings.Join(lines, "\n"))) {
			t.Errorf("%s is not valid JSON under its comments", name)
		}
	}
}
