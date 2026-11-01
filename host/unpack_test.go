package host_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/zadykian/ws/host"
)

type entry struct {
	name, body, link string
	typ              byte
	mode             int64
}

func archive(t *testing.T, zipped bool, entries ...entry) []byte {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range entries {
		if e.typ == 0 {
			e.typ = tar.TypeReg
		}
		hdr := &tar.Header{Name: e.name, Typeflag: e.typ, Mode: e.mode, Linkname: e.link}
		if e.typ == tar.TypeReg {
			hdr.Size = int64(len(e.body))
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(e.body)); err != nil && e.typ == tar.TypeReg {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if !zipped {
		return buf.Bytes()
	}
	var out bytes.Buffer
	zw := gzip.NewWriter(&out)
	_, _ = zw.Write(buf.Bytes())
	_ = zw.Close()
	return out.Bytes()
}

func unpack(t *testing.T, data []byte, strip int) (string, error) {
	dir := filepath.Join(t.TempDir(), "tree")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	return dir, host.Unpack(bytes.NewReader(data), root, strip)
}

func TestUnpackStripsTheTopDirectory(t *testing.T) {
	for _, zipped := range []bool{true, false} {
		data := archive(t, zipped,
			entry{name: "go/", typ: tar.TypeDir, mode: 0o755},
			entry{name: "go/VERSION", body: "go1.27.3\n", mode: 0o644},
			entry{name: "go/bin/", typ: tar.TypeDir, mode: 0o755},
			entry{name: "go/bin/go", body: "#!", mode: 0o755},
		)
		dir, err := unpack(t, data, 1)
		if err != nil {
			t.Fatalf("zipped %v: %v", zipped, err)
		}
		got, err := os.ReadFile(filepath.Join(dir, "VERSION"))
		if err != nil || string(got) != "go1.27.3\n" {
			t.Errorf("zipped %v: VERSION is %q, %v", zipped, got, err)
		}
		if fi, err := os.Stat(filepath.Join(dir, "bin", "go")); err != nil || fi.Mode().Perm() != 0o755 {
			t.Errorf("zipped %v: bin/go is %v, %v", zipped, fi, err)
		}
		if _, err := os.Stat(filepath.Join(dir, "go")); err == nil {
			t.Errorf("zipped %v: the top directory was kept", zipped)
		}
	}
}

func TestUnpackKeepsTheNamesWithNoStrip(t *testing.T) {
	dir, err := unpack(t, archive(t, true, entry{name: "a/b", body: "x", mode: 0o600}), 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a", "b")); err != nil {
		t.Error(err)
	}
}
