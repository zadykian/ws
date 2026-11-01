package host_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
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
			entry{name: "go/bin/gofmt", typ: tar.TypeSymlink, link: "go"},
			entry{name: "go/bin/again", typ: tar.TypeLink, link: "go/bin/go"},
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
		if l, err := os.Readlink(filepath.Join(dir, "bin", "gofmt")); err != nil || l != "go" {
			t.Errorf("zipped %v: gofmt links to %q, %v", zipped, l, err)
		}
		if b, err := os.ReadFile(filepath.Join(dir, "bin", "again")); err != nil || string(b) != "#!" {
			t.Errorf("zipped %v: the hard link reads %q, %v", zipped, b, err)
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

func TestUnpackRefuses(t *testing.T) {
	tests := []struct {
		name    string
		entries []entry
		want    string
	}{
		{"an absolute name", []entry{{name: "/etc/x", body: "x"}}, "absolute"},
		{"a name that goes up", []entry{{name: "a/../../x", body: "x"}}, "leaves the directory"},
		{"a name through a link out", []entry{
			{name: "a/l", typ: tar.TypeSymlink, link: "/tmp"},
			{name: "a/l/x", body: "x"},
		}, ""},
		{"a link out of the tree", []entry{
			{name: "a/l", typ: tar.TypeSymlink, link: "../.."},
			{name: "a/l/x", body: "x"},
		}, ""},
		{"a hard link to a name that goes up", []entry{{name: "a/h", typ: tar.TypeLink, link: "../x"}}, "leaves"},
		{"a device", []entry{{name: "a/d", typ: tar.TypeChar}}, "type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := unpack(t, archive(t, true, tt.entries...), 1)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want an error with %q", err, tt.want)
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "x")); err == nil {
				t.Error("a file was written beside the tree")
			}
		})
	}
}

func TestUnpackFailsOnACutArchive(t *testing.T) {
	data := archive(t, true, entry{name: "a/b", body: strings.Repeat("x", 5000)})
	if _, err := unpack(t, data[:len(data)/2], 1); err == nil {
		t.Error("a cut archive unpacked")
	}
	if _, err := unpack(t, []byte("not a tarball at all, 512 bytes needed"), 1); err == nil {
		t.Error("garbage unpacked")
	}
}

func TestUnpackFile(t *testing.T) {
	file := filepath.Join(t.TempDir(), "a.tar.gz")
	data := archive(t, true, entry{name: "top/f", body: "y", mode: 0o644})
	if err := os.WriteFile(file, data, 0o600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "new", "tree")
	if err := host.UnpackFile(file, dir, 1); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "f")); err != nil || string(b) != "y" {
		t.Errorf("f is %q, %v", b, err)
	}
	if err := host.UnpackFile(filepath.Join(dir, "missing"), dir, 1); err == nil {
		t.Error("a missing archive unpacked")
	}
}
