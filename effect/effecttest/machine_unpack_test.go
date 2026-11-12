package effecttest_test

import (
	"archive/tar"
	"bytes"
	"context"
	"io/fs"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

type member struct {
	name, body, link string
	typ              byte
	mode             int64
}

func tarball(t *testing.T, members ...member) []byte {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for _, e := range members {
		hdr := &tar.Header{Name: e.name, Typeflag: e.typ, Mode: e.mode, Linkname: e.link}
		if e.typ == tar.TypeReg {
			hdr.Size = int64(len(e.body))
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if e.typ == tar.TypeReg {
			if _, err := tw.Write([]byte(e.body)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestMachineUnpackRecordsTheTree(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/c/go.tgz"] = tarball(t,
		member{name: "go/", typ: tar.TypeDir, mode: 0o755},
		member{name: "go/VERSION", typ: tar.TypeReg, body: "go1.27\n", mode: 0o644},
		member{name: "go/bin/", typ: tar.TypeDir, mode: 0o750},
		member{name: "go/bin/go", typ: tar.TypeReg, body: "#!", mode: 0o755},
		member{name: "go/bin/gofmt", typ: tar.TypeSymlink, link: "go"},
	)
	if err := m.Unpack(context.Background(), "/c/go.tgz", "/usr/local/go", 1); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/usr/local/go/VERSION"]); got != "go1.27\n" {
		t.Errorf("VERSION is %q", got)
	}
	if m.Dirs["/usr/local/go"] != 0o755 || m.Dirs["/usr/local/go/bin"] != 0o750 ||
		m.Modes["/usr/local/go/bin/go"] != fs.FileMode(0o755) || m.Links["/usr/local/go/bin/gofmt"] != "go" {
		t.Errorf("Dirs %v, Modes %v, Links %v", m.Dirs, m.Modes, m.Links)
	}
	if want := []string{"unpack /c/go.tgz /usr/local/go"}; !slices.Equal(m.Calls, want) {
		t.Errorf("Calls = %q, want %q", m.Calls, want)
	}
}
