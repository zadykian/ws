package effect_test

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect"
)

func TestOSMachineFiles(t *testing.T) {
	m := effect.OSMachine{Sudo: []string{"env"}, Owner: fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())}
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
	m := effect.OSMachine{Sudo: []string{"env"}, Owner: fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())}
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
	if err := (effect.OSMachine{Sudo: []string{"false"}}).WriteRootFile(path, nil, 0o600); err == nil {
		t.Error("WriteRootFile through a failing sudo: err = nil")
	}
}
