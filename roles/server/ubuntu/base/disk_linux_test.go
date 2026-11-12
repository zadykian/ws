//go:build linux

package base

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOSDiskStatfs(t *testing.T) {
	if got, err := (osDisk{}).Statfs(t.TempDir()); err != nil || got <= 0 {
		t.Errorf("Statfs = %d, %v, want a positive number", got, err)
	}
	if _, err := (osDisk{}).Statfs(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("Statfs of a missing path did not fail")
	}
}

func TestOSDiskRestrict(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("chown to root needs root")
	}
	path := filepath.Join(t.TempDir(), "f")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := (osDisk{}).Restrict(path); err != nil {
		t.Fatal(err)
	}
	if info, _ := (osDisk{}).Stat(path); info.Mode.Perm() != 0o600 || info.UID != 0 {
		t.Errorf("Stat = %+v, want mode 0600 and root", info)
	}
}
