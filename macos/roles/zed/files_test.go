package zed_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestTheZedFilesAreJSONUnderTheirComments(t *testing.T) {
	for _, name := range []string{"global_settings.json", "keymap.json"} {
		file, err := os.ReadFile("files/" + name) //nolint:gosec // a file of the role, named above
		if err != nil {
			t.Fatal(err)
		}
		var lines []string
		for l := range strings.Lines(string(file)) {
			if !strings.HasPrefix(strings.TrimSpace(l), "//") {
				lines = append(lines, l)
			}
		}
		if !json.Valid([]byte(strings.Join(lines, ""))) {
			t.Errorf("%s is not valid JSON under its comments", name)
		}
	}
}
