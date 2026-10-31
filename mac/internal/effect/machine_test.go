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

func TestOSMachineRunShownAndPass(t *testing.T) {
	ctx := context.Background()
	var stderr strings.Builder
	m := effect.OSMachine{Sudo: []string{"env", "WS_TEST=1"}, Stderr: &stderr}
	if out, err := m.RunShown(ctx, false, "sh", "-c", "echo out; echo err >&2"); err != nil || string(out) != "out\n" {
		t.Errorf("RunShown = %q, %v; want out", out, err)
	}
	if stderr.String() != "err\n" {
		t.Errorf("RunShown passed %q, want err", stderr.String())
	}
	stderr.Reset()
	if err := m.Pass(ctx, true, "sh", "-c", "echo out; echo $WS_TEST >&2"); err != nil {
		t.Errorf("Pass: %v", err)
	}
	if stderr.String() != "out\n1\n" {
		t.Errorf("Pass passed %q, want out and 1", stderr.String())
	}
	if err := m.Pass(ctx, false, "false"); err == nil {
		t.Error("Pass of a failing command: no error")
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

func TestOSMachineReadDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "b"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if names, err := (effect.OSMachine{}).ReadDir(dir); err != nil || len(names) != 1 || names[0] != "b" {
		t.Errorf("ReadDir = %v, %v; want [b]", names, err)
	}
}

func TestOSMachineMkdirChmod(t *testing.T) {
	m := effect.OSMachine{}
	dir := filepath.Join(t.TempDir(), "d")
	if err := m.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := m.Chmod(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if info, err := m.Stat(dir); err != nil || info.Mode.Perm() != 0o750 {
		t.Errorf("Stat after Mkdir and Chmod = %+v, %v; want 750", info, err)
	}
}
