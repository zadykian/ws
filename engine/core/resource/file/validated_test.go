package file_test

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
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

func TestValidatedSecretHidesTheOutput(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/srv/d.env"] = []byte("old")
	m.Results["check --file /srv/d.env.ws-check"] = effecttest.Result{Out: []byte("bad VALUE\n"), Err: errors.New("exit 1")}
	v := file.Validated{
		File:  file.File{M: m, Path: "/srv/d.env", Content: []byte("new"), Mode: 0o600, Secret: true},
		Check: func(path string) (string, []string) { return "check", []string{"--file", path} },
	}
	g := resource.Group{Title: "a", Parts: []resource.Resource{v}}
	p, err := g.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	err = g.Apply(t.Context(), p)
	if err == nil || strings.Contains(err.Error(), "VALUE") || !strings.Contains(err.Error(), "check refuses the new /srv/d.env") {
		t.Fatalf("apply = %v, want a refusal that names the checker and the path, not the output", err)
	}
	if string(m.Files["/srv/d.env"]) != "old" {
		t.Errorf("file = %q, want old", m.Files["/srv/d.env"])
	}
	if _, ok := m.Files["/srv/d.env.ws-check"]; ok {
		t.Error("the checked copy was left behind")
	}
}

// The real machine's error quotes the checker's stderr, which may hold the content.
func TestValidatedSecretHidesTheCheckersStderr(t *testing.T) {
	_, err := exec.CommandContext(t.Context(), "sh", "-c", "echo VALUE >&2; exit 3").Output()
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("sh = %v, want an exit error", err)
	}
	m := effecttest.NewMachine()
	m.Files["/srv/d.env"] = []byte("old")
	m.Results["check --file /srv/d.env.ws-check"] = effecttest.Result{
		Err: fmt.Errorf("[check --file /srv/d.env.ws-check]: %w: %s", ee, strings.TrimSpace(string(ee.Stderr))),
	}
	v := file.Validated{
		File:  file.File{M: m, Path: "/srv/d.env", Content: []byte("new"), Mode: 0o600, Secret: true},
		Check: func(path string) (string, []string) { return "check", []string{"--file", path} },
	}
	g := resource.Group{Title: "a", Parts: []resource.Resource{v}}
	p, err := g.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	err = g.Apply(t.Context(), p)
	if err == nil || strings.Contains(err.Error(), "VALUE") || !strings.Contains(err.Error(), "check refuses the new /srv/d.env: exit status 3") {
		t.Fatalf("apply = %v, want a refusal with the exit status and none of the stderr", err)
	}
	if string(m.Files["/srv/d.env"]) != "old" {
		t.Errorf("file = %q, want old", m.Files["/srv/d.env"])
	}
}
