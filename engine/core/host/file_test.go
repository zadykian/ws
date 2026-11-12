package host_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host"
)

func applyAll(t *testing.T, r host.Resource) []string {
	changes, err := r.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var shown []string
	for _, c := range changes {
		shown = append(shown, c.String())
		if err := c.Apply(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	return shown
}

func TestFile(t *testing.T) {
	m := effecttest.NewMachine()
	f := host.File{M: m, Path: "/srv/a", Content: []byte("new\n"), Mode: 0o600, UID: 5, GID: 6}
	if got := applyAll(t, f); len(got) != 1 || !strings.HasPrefix(got[0], "create /srv/a") {
		t.Errorf("first plan = %q, want a create", got)
	}
	if string(m.Files["/srv/a"]) != "new\n" || m.Calls[len(m.Calls)-1] != "atomic /srv/a 600" {
		t.Errorf("file = %q, calls %v", m.Files["/srv/a"], m.Calls)
	}
	if got := applyAll(t, f); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
	f.Content = []byte("newer\n")
	if got := applyAll(t, f); len(got) != 1 || !strings.Contains(got[0], "-new\n+newer") {
		t.Errorf("content plan = %q, want a diff", got)
	}
	f.Mode = 0o644
	if got := applyAll(t, f); len(got) != 1 || !strings.HasPrefix(got[0], "set owner 5:6 and mode 644 of /srv/a") {
		t.Errorf("mode plan = %q", got)
	}
}

func TestFileSecretShowsNoDiff(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/k"] = []byte("old-secret")
	f := host.File{M: m, Path: "/k", Content: []byte("new-secret"), Mode: 0o600, Secret: true}
	got := applyAll(t, f)
	if len(got) != 1 || strings.Contains(got[0], "secret") || !strings.Contains(got[0], "content not shown") {
		t.Errorf("plan = %q, want no content", got)
	}
}

func TestFileBackupKeepsTheOldFileOnce(t *testing.T) {
	defer host.FixBackupName("1")()
	m := effecttest.NewMachine()
	m.Files["/srv/a"] = []byte("old\n")
	m.Modes["/srv/a"], m.Owners["/srv/a"] = 0o640, [2]int{5, 6}
	f := host.File{M: m, Path: "/srv/a", Content: []byte("new\n"), Mode: 0o600, UID: 5, GID: 6, Backup: true}
	applyAll(t, f)
	if got := string(m.Files["/srv/a.1~"]); got != "old\n" {
		t.Errorf("backup = %q, want the old content", got)
	}
	delete(m.Files, "/srv/a.1~")
	if got := applyAll(t, f); got != nil || len(m.Files) != 1 {
		t.Errorf("second plan = %q, files %d; want none and no backup", got, len(m.Files))
	}
	f.Path = "/srv/b"
	applyAll(t, f)
	if _, ok := m.Files["/srv/b.1~"]; ok {
		t.Error("a new file has a backup")
	}
}
