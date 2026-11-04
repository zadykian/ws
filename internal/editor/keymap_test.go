package editor

import (
	"io/fs"
	"slices"
	"strings"
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
	if len(keymap[Global]) == 0 || len(keymap[Text]) == 0 {
		t.Errorf("empty scopes: %d, %d", len(keymap[Global]), len(keymap[Text]))
	}
}

func TestAKeyTwiceInAScopeFails(t *testing.T) {
	key, _ := ParseChord("ctrl+a")
	keymap := Keymap{Text: {{Action: "edit.one", Keys: []Chord{key}}, {Action: "edit.two", Keys: []Chord{key}}}}
	if err := keymap.validate(); err == nil || !strings.Contains(err.Error(), "binds both") {
		t.Errorf("error %v", err)
	}
}

func TestAKeyInTwoScopesPasses(t *testing.T) {
	key, _ := ParseChord("ctrl+a")
	keymap := Keymap{Text: {{Action: "edit.one", Keys: []Chord{key}}}, Tree: {{Action: "tree.one", Keys: []Chord{key}}}}
	if err := keymap.validate(); err != nil {
		t.Error(err)
	}
}

func TestAMisnamedActionFails(t *testing.T) {
	if err := (Keymap{Text: {{Action: "Edit Copy"}}}).validate(); err == nil {
		t.Error("no error")
	}
}
