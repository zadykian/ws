package effect_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

func tarGz(t *testing.T, names ...string) string {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	for _, name := range names {
		if err := tw.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: 0o644, Size: 1}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "a.tar.gz")
	if err := os.WriteFile(file, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestOSMachineUnpackStripsAndMakesTheDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new")
	err := effect.OSMachine{}.Unpack(context.Background(), tarGz(t, "top/a/f"), dir, 1)
	if err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "a", "f")); err != nil || string(b) != "x" { //nolint:gosec // a path in the test's own temp tree
		t.Errorf("a/f is %q, %v", b, err)
	}
	if fi, err := os.Stat(dir); err != nil || fi.Mode().Perm() != 0o755 {
		t.Errorf("dir is %v, %v", fi, err)
	}
}

func TestUnpackFileThroughTheMachine(t *testing.T) {
	file := tarGz(t, "top/f")
	if err := host.UnpackFile(file, filepath.Join(t.TempDir(), "d"), 1); err != nil {
		t.Error(err)
	}
	if err := host.UnpackFile(file+"-none", t.TempDir(), 1); err == nil {
		t.Error("a missing archive unpacked")
	}
}

func TestOSMachineUnpackRefusesAnEntryThatLeavesTheDirectory(t *testing.T) {
	base := t.TempDir()
	err := effect.OSMachine{}.Unpack(context.Background(), tarGz(t, "top/../../x"), filepath.Join(base, "d"), 1)
	if err == nil {
		t.Fatal("a name that goes up unpacked")
	}
	if _, err := os.Stat(filepath.Join(base, "x")); err == nil {
		t.Error("a file was written beside the directory")
	}
}

func TestOSMachineUnpackRefusesALinkThatLeavesTheDirectory(t *testing.T) {
	for _, target := range []string{"/etc", "../x"} {
		var buf bytes.Buffer
		zw := gzip.NewWriter(&buf)
		tw := tar.NewWriter(zw)
		if err := tw.WriteHeader(&tar.Header{Name: "top/l", Typeflag: tar.TypeSymlink, Linkname: target}); err != nil {
			t.Fatal(err)
		}
		if err := errors.Join(tw.Close(), zw.Close()); err != nil {
			t.Fatal(err)
		}
		file, dir := filepath.Join(t.TempDir(), "a.tar.gz"), filepath.Join(t.TempDir(), "d")
		if err := os.WriteFile(file, buf.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := (effect.OSMachine{}).Unpack(context.Background(), file, dir, 1); err == nil {
			t.Errorf("a link to %s unpacked", target)
		}
		if _, err := os.Lstat(filepath.Join(dir, "l")); err == nil {
			t.Errorf("a link to %s was made", target)
		}
	}
}
