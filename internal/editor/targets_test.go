package editor

import (
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
