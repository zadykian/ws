package zed_test

import (
	"bytes"
	"embed"
	"encoding/json"
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/editor/zed"
)

//go:embed testdata/*.json
var goldens embed.FS

// golden fails where the render differs. A meant change edits the file by hand, in its commit.
func golden(t *testing.T, name string, got []byte) {
	want, err := goldens.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s differs: %d bytes, want %d\n%s", name, len(got), len(want), got)
	}
}

func TestKeymapMatchesTheGoldenFileByteForByte(t *testing.T) {
	got, err := zed.Keymap()
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "keymap.json", got)
}

func TestTheKeymapHasWsMarkAndIsJSONUnderItsComments(t *testing.T) {
	got, err := zed.Keymap()
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	for _, l := range strings.Split(string(got), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(l), "//") {
			lines = append(lines, l)
		}
	}
	if !strings.HasPrefix(string(got), "// Written by github.com/zadykian/ws") {
		t.Error("the keymap does not start with ws's mark")
	}
	if !json.Valid([]byte(strings.Join(lines, "\n"))) {
		t.Error("the keymap is not valid JSON under its comments")
	}
}

func TestTheSettingsAreJSON(t *testing.T) {
	got, err := zed.Settings()
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(got) {
		t.Errorf("the settings are not valid JSON:\n%s", got)
	}
}
