package local_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/local"
)

func TestOSMachineFiles(t *testing.T) {
	m := local.OSMachine{Sudo: []string{"env"}, Owner: fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())}
	path := filepath.Join(t.TempDir(), "f")
	if _, err := m.ReadFile(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("ReadFile of a missing file: err = %v, want fs.ErrNotExist", err)
	}
	for _, write := range []func(string, []byte, fs.FileMode) error{m.WriteFile, m.WriteRootFile} {
		if err := write(path, []byte("one"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := write(path, []byte("two"), 0o640); err != nil {
			t.Fatal(err)
		}
		got, err := m.ReadFile(path)
		if err != nil || string(got) != "two" {
			t.Errorf("ReadFile = %q, %v; want two", got, err)
		}
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0o640 {
			t.Errorf("mode = %v, %v; want 0640", info.Mode(), err)
		}
	}
}

func TestOSMachineRootFiles(t *testing.T) {
	m := local.OSMachine{Sudo: []string{"env"}, Owner: fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())}
	dir := t.TempDir()
	path := filepath.Join(dir, "f")
	if _, err := m.RootAttrs(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("RootAttrs of a missing file: err = %v, want fs.ErrNotExist", err)
	}
	if _, err := m.ReadRootFile(path); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("ReadRootFile of a missing file: err = %v, want fs.ErrNotExist", err)
	}
	if err := m.WriteRootFile(path, []byte("key"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := m.ReadRootFile(path)
	if err != nil || string(got) != "key" {
		t.Errorf("ReadRootFile = %q, %v; want key", got, err)
	}
	attrs, err := m.RootAttrs(path)
	want := effect.Attrs{UID: os.Getuid(), GID: os.Getgid(), Mode: 0o600}
	if err != nil || attrs != want {
		t.Errorf("RootAttrs = %+v, %v; want %+v", attrs, err, want)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("the directory holds %d files, want the target alone", len(entries))
	}
	if err := (local.OSMachine{Sudo: []string{"false"}}).WriteRootFile(path, nil, 0o600); err == nil {
		t.Error("WriteRootFile through a failing sudo: err = nil")
	}
}

// userReader reads root's files as the test's user, with a sudo that fails, so that a read through
// it would fail the test.
var userReader = local.OSMachine{Sudo: []string{"false"}, UserReads: true}

func TestUserReadsStatsAsTheUser(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, []byte("text"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o400); err != nil {
		t.Fatal(err)
	}
	attrs, err := userReader.RootAttrs(path)
	want := effect.Attrs{UID: os.Getuid(), GID: os.Getgid(), Mode: 0o400}
	if err != nil || attrs != want {
		t.Errorf("RootAttrs = %+v, %v; want %+v", attrs, err, want)
	}
	got, err := userReader.ReadRootFile(path)
	if err != nil || string(got) != "text" {
		t.Errorf("ReadRootFile = %q, %v; want text", got, err)
	}
}

func TestUserReadsMissing(t *testing.T) {
	dir := t.TempDir()
	if err := os.Symlink("nowhere", filepath.Join(dir, "dangling")); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"missing", "dangling"} {
		path := filepath.Join(dir, name)
		if _, err := userReader.RootAttrs(path); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("RootAttrs(%s): err = %v, want fs.ErrNotExist", name, err)
		}
		if _, err := userReader.ReadRootFile(path); !errors.Is(err, fs.ErrNotExist) {
			t.Errorf("ReadRootFile(%s): err = %v, want fs.ErrNotExist", name, err)
		}
	}
}

func TestUserReadsRefusesUnreadable(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads every file")
	}
	dir := filepath.Join(t.TempDir(), "closed")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "f")
	if err := os.WriteFile(path, []byte("key"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) }) //nolint:gosec // the test's own directory, searchable again so that TempDir removes it
	for name, read := range map[string]func(string) error{
		"RootAttrs":    func(p string) error { _, err := userReader.RootAttrs(p); return err },
		"ReadRootFile": func(p string) error { _, err := userReader.ReadRootFile(p); return err },
	} {
		err := read(path)
		if !errors.Is(err, fs.ErrPermission) || !strings.Contains(err.Error(), "not readable by ws's user") {
			t.Errorf("%s of a closed file: err = %v, want the refusal", name, err)
		}
	}
}

func TestReadsGoThroughSudo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, []byte("text"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := local.OSMachine{Sudo: []string{"false"}}
	if _, err := m.RootAttrs(path); err == nil {
		t.Error("RootAttrs through a failing sudo: err = nil")
	}
	if _, err := m.ReadRootFile(path); err == nil {
		t.Error("ReadRootFile through a failing sudo: err = nil")
	}
}
