package editor

import (
	"io/fs"
	"slices"
	"testing"
)

func TestTopicsAreTheEmbeddedFiles(t *testing.T) {
	names, err := fs.Glob(keymapFiles, "keymap/*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	want := make([]string, len(topics))
	for i, topic := range topics {
		want[i] = "keymap/" + topic + ".yaml"
	}
	slices.Sort(want)
	if !slices.Equal(names, want) {
		t.Errorf("files %v, topics %v", names, want)
	}
}

func TestTheKeymapLoads(t *testing.T) {
	keymap, err := LoadKeymap()
	if err != nil {
		t.Fatal(err)
	}
	if len(keymap[Global]) == 0 || len(keymap[Code]) == 0 {
		t.Errorf("empty scopes: %d, %d", len(keymap[Global]), len(keymap[Code]))
	}
}
