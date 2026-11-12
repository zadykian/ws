//go:build linux

package base

import (
	"io"
	"path/filepath"
	"testing"
)

func TestOSDiskAllocate(t *testing.T) {
	d := osDisk{}
	path := filepath.Join(t.TempDir(), "swapfile")
	if info, err := d.Stat(path); err != nil || info.Exists {
		t.Fatalf("Stat of a missing path = %+v, %v", info, err)
	}
	if err := d.Allocate(path, 3); err != nil {
		t.Fatal(err)
	}
	info, err := d.Stat(path)
	if err != nil || !info.Exists || info.Size != 3*mib || info.Mode.Perm() != 0o600 {
		t.Errorf("Stat = %+v, %v, want 3 MiB, mode 0600", info, err)
	}
	if err := d.Allocate(path, 3); err == nil {
		t.Error("Allocate over a file did not fail")
	}
	if info, _ := d.Stat(path); info.Size != 3*mib {
		t.Errorf("the failed Allocate changed the file: %+v", info)
	}
	moved := path + "x"
	if err := d.Rename(path, moved); err != nil {
		t.Fatal(err)
	}
	if err := d.Remove(moved); err != nil {
		t.Fatal(err)
	}
	if err := d.Remove(moved); err != nil {
		t.Errorf("Remove of a missing file: %v", err)
	}
}

func TestZeros(t *testing.T) {
	got, err := io.ReadAll(io.LimitReader(zeros{}, 70000))
	if err != nil || len(got) != 70000 {
		t.Fatalf("read %d bytes, %v", len(got), err)
	}
	for i, b := range got {
		if b != 0 {
			t.Fatalf("byte %d is %d", i, b)
		}
	}
}
