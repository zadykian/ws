package effect_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect"
)

func TestOSMachineRun(t *testing.T) {
	ctx := context.Background()
	m := effect.OSMachine{Sudo: []string{"env", "WS_TEST=1"}}
	if out, err := m.Run(ctx, "echo", "hi"); err != nil || string(out) != "hi\n" {
		t.Errorf("Run = %q, %v; want hi", out, err)
	}
	if out, err := m.RunRoot(ctx, "sh", "-c", "echo $WS_TEST"); err != nil || string(out) != "1\n" {
		t.Errorf("RunRoot = %q, %v; want 1", out, err)
	}
	_, err := m.Run(ctx, "sh", "-c", "echo boom >&2; exit 3")
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("Run of a failing command: err = %v, want its stderr", err)
	}
}

func TestOSMachineStat(t *testing.T) {
	m := effect.OSMachine{}
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("sub", filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	info, err := m.Stat(sub)
	if err != nil || !info.Mode.IsDir() || info.Mode.Perm() != 0o700 || info.Link != "" {
		t.Errorf("Stat(sub) = %+v, %v; want a directory, 700", info, err)
	}
	if info, err := m.Stat(filepath.Join(dir, "link")); err != nil || info.Link != "sub" || !info.Mode.IsDir() {
		t.Errorf("Stat(link) = %+v, %v; want sub's mode and its link", info, err)
	}
	if _, err := m.Stat(filepath.Join(dir, "missing")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Stat(missing) err = %v, want ErrNotExist", err)
	}
}
