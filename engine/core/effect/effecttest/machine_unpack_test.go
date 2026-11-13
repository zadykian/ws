package effecttest_test

import (
	"archive/tar"
	"bytes"
	"context"
	"io/fs"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
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
func TestMachineUnpackRefuses(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want string
	}{
		{"a name that goes up", tarball(t, member{name: "a/../../x", typ: tar.TypeReg, body: "x"}), "leaves the directory"},
		{"a link out of the tree", tarball(t, member{name: "a/l", typ: tar.TypeSymlink, link: "/etc"}), "leaves the directory"},
		{"a missing archive", nil, "file does not exist"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := effecttest.NewMachine()
			if tt.data != nil {
				m.Files["/c/a.tgz"] = tt.data
			}
			err := m.Unpack(context.Background(), "/c/a.tgz", "/d", 1)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want an error with %q", err, tt.want)
			}
			if len(m.Files) > 1 || len(m.Links) != 0 {
				t.Errorf("a refused archive left %v and %v", m.Files, m.Links)
			}
		})
	}
}
