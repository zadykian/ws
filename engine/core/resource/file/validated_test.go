package file_test

import (
	"errors"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
)

func validatedFile(m *effecttest.Machine) file.Validated {
	return file.Validated{
		File: file.File{M: m, Path: "/srv/d.json", Content: []byte("new"), Mode: 0o644},
		Check: func(path string) (string, []string) {
			return "check", []string{"--file", path}
		},
	}
}

func TestValidatedWritesAGoodFile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/srv/d.json"] = []byte("old")
	m.Results["check --file /srv/d.json.ws-check"] = effecttest.Result{}
	if got := applyAll(t, validatedFile(m)); len(got) != 1 {
		t.Fatalf("plan = %q, want one write", got)
	}
	if string(m.Files["/srv/d.json"]) != "new" {
		t.Errorf("file = %q, want new", m.Files["/srv/d.json"])
	}
	if _, ok := m.Files["/srv/d.json.ws-check"]; ok {
		t.Error("the checked copy was left behind")
	}
}

func TestValidatedKeepsTheOldFileOnARefusal(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/srv/d.json"] = []byte("old")
	m.Results["check --file /srv/d.json.ws-check"] = effecttest.Result{Out: []byte("bad key\n"), Err: errors.New("exit 1")}
	g := resource.Group{Title: "a", Parts: []resource.Resource{validatedFile(m)}}
	p, err := g.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if err := g.Apply(t.Context(), p); err == nil {
		t.Fatal("a refused file was written")
	}
	if string(m.Files["/srv/d.json"]) != "old" {
		t.Errorf("file = %q, want old", m.Files["/srv/d.json"])
	}
}
