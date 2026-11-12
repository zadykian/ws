package host_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zadykian/ws/host"
)

func TestUnpackFile(t *testing.T) {
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(zw)
	if err := tw.WriteHeader(&tar.Header{Name: "top/f", Typeflag: tar.TypeReg, Mode: 0o644, Size: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("y")); err != nil {
		t.Fatal(err)
	}
	if err := errors.Join(tw.Close(), zw.Close()); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "a.tar.gz")
	if err := os.WriteFile(file, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "new", "tree")
	if err := host.UnpackFile(file, dir, 1); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "f")); err != nil || string(b) != "y" { //nolint:gosec // a path in the test's own temp tree
		t.Errorf("f is %q, %v", b, err)
	}
	if err := host.UnpackFile(filepath.Join(dir, "missing"), dir, 1); err == nil {
		t.Error("a missing archive unpacked")
	}
}
