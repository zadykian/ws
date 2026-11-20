package secrets_test

import (
	"os"
	"path/filepath"
	"testing"
)

// fetched runs the task once, so that the clone exists, and clears mise's log of that run.
func (f *fixture) fetched() {
	if code, out := f.run(); code != 0 {
		f.t.Fatalf("the first run exits %d:\n%s", code, out)
	}
	if err := os.Remove(filepath.Join(f.logs, "mise")); err != nil {
		f.t.Fatal(err)
	}
}

// fnox loads every fnox file of the clone, an ignored one too, so the task runs on origin's main
// alone. Each refusal names the clone, and a run after its removal clones it again.
func TestTaskRefusesAChangeOfTheClones(t *testing.T) {
	for _, c := range []struct {
		name   string
		want   string
		change func(f *fixture)
	}{
		{"a tracked change", "holds changes of its own", func(f *fixture) { f.write("fnox.toml", "root = false\n") }},
		{"an untracked file", "holds changes of its own", func(f *fixture) { f.write("fnox.other.toml", "") }},
		{"an ignored fnox.local.toml", "holds changes of its own", func(f *fixture) {
			f.write(".git/info/exclude", "fnox.local.toml\n")
			f.write("fnox.local.toml", "[secrets]\n")
		}},
		{"a commit of its own", "holds a commit of its own", func(f *fixture) {
			f.git(f.clone(), "commit", "-q", "--allow-empty", "-m", "local")
		}},
		{"a detached HEAD", "is not on main", func(f *fixture) { f.git(f.clone(), "checkout", "-q", "--detach") }},
		{"another branch", "is not on main", func(f *fixture) { f.git(f.clone(), "checkout", "-q", "-b", "other") }},
		{"no clone", "is not a clone", func(f *fixture) {
			if err := os.RemoveAll(filepath.Join(f.clone(), ".git")); err != nil {
				f.t.Fatal(err)
			}
		}},
		{"a link", "is not a clone", func(f *fixture) {
			other := filepath.Join(f.root, "other")
			if err := os.Rename(f.clone(), other); err != nil {
				f.t.Fatal(err)
			}
			if err := os.Symlink(other, f.clone()); err != nil {
				f.t.Fatal(err)
			}
		}},
	} {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			f.fetched()
			c.change(f)
			code, out := f.run()
			f.refused(code, out, f.clone()+" "+c.want+": remove it")
			if err := os.RemoveAll(f.clone()); err != nil {
				t.Fatal(err)
			}
			if code, out := f.run(); code != 0 {
				t.Fatalf("a run after the clone's removal exits %d:\n%s", code, out)
			}
			f.cloned()
		})
	}
}

// A fetch without + refuses a main that origin rewrote, so a forced push reaches no server.
func TestTaskRefusesARewrittenOrigin(t *testing.T) {
	f := newFixture(t)
	f.fetched()
	f.git(f.seed, "checkout", "-q", "--orphan", "other")
	f.git(f.seed, "commit", "-q", "-m", "other")
	f.git(f.seed, "push", "-q", "-f", "origin", "other:main")
	code, out := f.run()
	f.refused(code, out, "has a main that origin no longer holds")
}

func TestTaskRefusesAnotherOrigin(t *testing.T) {
	f := newFixture(t)
	f.fetched()
	f.git(f.clone(), "remote", "set-url", "origin", filepath.Join(f.root, "elsewhere.git"))
	code, out := f.run()
	f.refused(code, out, "another repository")
}

// write writes a file of the clone's.
func (f *fixture) write(name, text string) {
	if err := os.WriteFile(filepath.Join(f.clone(), name), []byte(text), 0o600); err != nil {
		f.t.Fatal(err)
	}
}
