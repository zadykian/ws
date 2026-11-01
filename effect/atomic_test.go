package effect_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zadykian/ws/effect"
)

func TestWriteAtomic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := effect.Attrs{UID: os.Getuid(), GID: os.Getgid(), Mode: 0o640}
	if err := (effect.OSMachine{}).WriteAtomic(path, []byte("new"), a); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path) //nolint:gosec // a temporary path of the test
	if err != nil || string(got) != "new" {
		t.Errorf("content = %q, %v; want new", got, err)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0o640 {
		t.Errorf("mode = %v, %v; want 0640", info.Mode(), err)
	}
	if left, _ := filepath.Glob(path + "*"); len(left) != 1 {
		t.Errorf("files beside the target: %v, want the target alone", left)
	}
}
