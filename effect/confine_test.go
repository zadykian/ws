package effect_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zadykian/ws/effect"
)

// TestConfine swaps a link in at the last component and at a parent, as ws_user could while root
// applies, and checks that each confined write fails and leaves the file outside as it was.
func TestConfine(t *testing.T) {
	attrs := effect.Attrs{UID: os.Getuid(), GID: os.Getgid(), Mode: 0o600}
	ops := map[string]func(m effect.OSMachine, p string) error{
		"Mkdir":       func(m effect.OSMachine, p string) error { return m.Mkdir(p, 0o700) },
		"Chmod":       func(m effect.OSMachine, p string) error { return m.Chmod(p, 0o777) },
		"Chown":       func(m effect.OSMachine, p string) error { return m.Chown(p, os.Getuid(), os.Getgid()) },
		"WriteFile":   func(m effect.OSMachine, p string) error { return m.WriteFile(p, []byte("x"), 0o600) },
		"WriteAtomic": func(m effect.OSMachine, p string) error { return m.WriteAtomic(p, []byte("x"), attrs) },
		"Symlink":     func(m effect.OSMachine, p string) error { return m.Symlink("t", p) },
	}
	for name, op := range ops {
		for _, where := range []string{"last", "parent"} {
			t.Run(name+"/"+where, func(t *testing.T) {
				home, out := t.TempDir(), t.TempDir()
				victim := filepath.Join(out, "f")
				if err := os.WriteFile(victim, []byte("keep"), 0o600); err != nil {
					t.Fatal(err)
				}
				target, link := victim, filepath.Join(home, "f")
				path := link
				if where == "parent" {
					target, link = out, filepath.Join(home, "d")
					path = filepath.Join(link, "f")
				}
				if err := os.Symlink(target, link); err != nil {
					t.Fatal(err)
				}
				// A rename replaces a link at the last component, and never follows it.
				renames := name == "Symlink" || name == "WriteAtomic"
				if err := op(effect.OSMachine{Confine: home}, path); err == nil && (where == "parent" || !renames) {
					t.Errorf("%s through %s succeeded", name, where)
				}
				got, _ := os.ReadFile(victim) //nolint:gosec // a temporary path of the test
				info, _ := os.Stat(victim)
				if string(got) != "keep" || info.Mode().Perm() != 0o600 {
					t.Errorf("outside file = %q %v, want keep 0600", got, info.Mode())
				}
				if left, _ := filepath.Glob(filepath.Join(out, "*")); len(left) != 1 {
					t.Errorf("outside holds %v, want the file alone", left)
				}
			})
		}
	}
}

func TestConfineAllows(t *testing.T) {
	home, out := t.TempDir(), t.TempDir()
	m := effect.OSMachine{Confine: home}
	if err := m.WriteAtomic(filepath.Join(out, "f"), []byte("a"), effect.Attrs{UID: os.Getuid(), GID: os.Getgid(), Mode: 0o600}); err != nil {
		t.Errorf("a path outside: %v", err)
	}
	if err := os.Mkdir(filepath.Join(home, "real"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("real", filepath.Join(home, "in")); err != nil {
		t.Fatal(err)
	}
	a := effect.Attrs{UID: os.Getuid(), GID: os.Getgid(), Mode: 0o640}
	if err := m.WriteAtomic(filepath.Join(home, "in", "f"), []byte("b"), a); err != nil {
		t.Fatalf("a link inside: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(home, "real", "f")) //nolint:gosec // a temporary path of the test
	if err != nil || string(got) != "b" {
		t.Errorf("content = %q, %v; want b", got, err)
	}
	if info, _ := os.Stat(filepath.Join(home, "real", "f")); info.Mode().Perm() != 0o640 {
		t.Errorf("mode = %v, want 0640", info.Mode())
	}
	if err := m.Symlink("real", filepath.Join(home, "l2")); err != nil {
		t.Errorf("Symlink: %v", err)
	}
}
