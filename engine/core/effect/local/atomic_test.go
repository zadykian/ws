package local_test

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/local"
)

func TestWriteAtomic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	a := effect.Attrs{UID: os.Getuid(), GID: os.Getgid(), Mode: 0o640}
	if err := (local.OSMachine{}).WriteAtomic(path, []byte("new"), a); err != nil {
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

func TestWriteNewMakesTheFileAndLeavesNothingBeside(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	a := effect.Attrs{UID: os.Getuid(), GID: os.Getgid(), Mode: 0o600}
	if err := (local.OSMachine{}).WriteNew(path, []byte("new"), a); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path) //nolint:gosec // a temporary path of the test
	info, statErr := os.Stat(path)
	if err != nil || string(got) != "new" || statErr != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("content = %q, %v; mode %v, %v; want new at 0600", got, err, info, statErr)
	}
	if left, _ := filepath.Glob(path + "*"); len(left) != 1 {
		t.Errorf("files beside the target: %v, want the target alone", left)
	}
}

func TestWriteNewReplacesNothing(t *testing.T) {
	dir := t.TempDir()
	file, link := filepath.Join(dir, "f"), filepath.Join(dir, "l")
	if err := os.WriteFile(file, []byte("old"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, "gone"), link); err != nil {
		t.Fatal(err)
	}
	a := effect.Attrs{UID: os.Getuid(), GID: os.Getgid(), Mode: 0o600}
	for _, path := range []string{file, link} {
		if err := (local.OSMachine{}).WriteNew(path, []byte("new"), a); !errors.Is(err, fs.ErrExist) {
			t.Errorf("WriteNew over %s = %v, want fs.ErrExist", filepath.Base(path), err)
		}
	}
	if got, _ := os.ReadFile(file); string(got) != "old" { //nolint:gosec // a temporary path of the test
		t.Errorf("file = %q, want old", got)
	}
	if left, _ := filepath.Glob(filepath.Join(dir, ".*")); len(left) != 0 {
		t.Errorf("pending files left: %v", left)
	}
}

func TestWriteNewRefusesAnotherOwner(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	err := (local.OSMachine{}).WriteNew(path, []byte("x"), effect.Attrs{UID: os.Geteuid() + 1, GID: 0, Mode: 0o600})
	if err == nil {
		t.Error("WriteNew gave another owner with no sudo")
	}
	if _, statErr := os.Lstat(path); statErr == nil {
		t.Error("the file was made")
	}
}

func TestWriteAtomicOfNoOwnerKeepsWsOwnOffSudo(t *testing.T) {
	m, log := loggedSudo(t)
	local.RunAs(t, os.Getuid()+1, false)
	t.Setenv("HOME", t.TempDir())
	path := filepath.Join(t.TempDir(), "f")
	if err := m.WriteAtomic(path, []byte("new"), effect.Attrs{UID: -1, GID: -1, Mode: 0o600}); err != nil {
		t.Fatal(err)
	}
	if got := sudoRan(t, log); got != "" {
		t.Errorf("sudo ran %q, want nothing", got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st, ok := info.Sys().(*syscall.Stat_t); !ok || int(st.Uid) != os.Getuid() || info.Mode().Perm() != 0o600 {
		t.Errorf("the file is %v, want ws's own and 0600", info)
	}
}

func TestSymlinkReplaces(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "l")
	if err := os.WriteFile(path, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := local.OSMachine{}
	for _, target := range []string{"/a", "/b"} {
		if err := m.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		if got, err := os.Readlink(path); err != nil || got != target {
			t.Errorf("Readlink = %q, %v; want %q", got, err, target)
		}
	}
}
