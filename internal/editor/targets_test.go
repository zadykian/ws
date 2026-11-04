package editor

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadTargetsFailsOnAnActionInTwoFiles(t *testing.T) {
	files := fstest.MapFS{
		"actions/a.yaml": {Data: []byte("edit.one: {to: a}\n")},
		"actions/b.yaml": {Data: []byte("edit.one: {to: b}\n")},
	}
	if _, err := LoadTargets(files, "actions/*.yaml"); err == nil {
		t.Error("no error")
	}
}

func TestLoadTargetsFailsOnAnUnknownField(t *testing.T) {
	files := fstest.MapFS{"actions/a.yaml": {Data: []byte("edit.one: {too: a}\n")}}
	if _, err := LoadTargets(files, "actions/*.yaml"); err == nil {
		t.Error("no error")
	}
}

func checked(targets map[string]Target) error {
	key, _ := ParseChord("ctrl+a")
	return Keymap{Text: {{Action: "edit.one", Keys: []Chord{key}}}}.Check(targets)
}

func TestCheckPassesATargetAndAReason(t *testing.T) {
	for _, target := range []Target{{To: "editor::One"}, {Why: "the editor has none"}} {
		if err := checked(map[string]Target{"edit.one": target}); err != nil {
			t.Error(err)
		}
	}
}

func TestCheckFailsOnAnActionWithNeither(t *testing.T) {
	for _, targets := range []map[string]Target{{}, {"edit.one": {}}} {
		if err := checked(targets); err == nil {
			t.Errorf("no error for %v", targets)
		}
	}
}

func TestCheckFailsOnATargetWithAReason(t *testing.T) {
	if err := checked(map[string]Target{"edit.one": {To: "a", Why: "b"}}); err == nil {
		t.Error("no error")
	}
}

func TestCheckFailsOnATargetOfNoAction(t *testing.T) {
	err := checked(map[string]Target{"edit.one": {To: "a"}, "edit.gone": {To: "b"}})
	if err == nil || !strings.Contains(err.Error(), "edit.gone") {
		t.Errorf("error %v", err)
	}
}
