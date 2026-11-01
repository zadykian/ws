package server

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

const (
	testLib  = "/var/lib/ws"
	testVars = "/etc/ws/host_vars.yml"
)

func testFS() fstest.MapFS {
	return fstest.MapFS{
		"site.yml":                       {Data: []byte("- hosts: all\n")},
		"requirements.yml":               {Data: []byte("- name: a.one\n  version: 1.0.0\n")},
		"collections/a-one-1.0.0.tar.gz": {Data: []byte("tar")},
		"roles/base/tasks/main.yml":      {Data: []byte("---\n")},
	}
}

func testPlaybook(t *testing.T, fsys fstest.MapFS, version string) (Playbook, *effecttest.Machine, string) {
	digest, err := Digest(fsys)
	if err != nil {
		t.Fatal(err)
	}
	m := effecttest.NewMachine()
	return Playbook{FS: fsys, Version: version, Lib: testLib, Vars: testVars, M: m, Out: &effecttest.Out{}}, m, digest
}

func TestPlaybookRefusals(t *testing.T) {
	bad := fstest.MapFS{"requirements.yml": {Data: []byte("- name: a.one\n  version: 2.0.0\n")},
		"collections/a-one-1.0.0.tar.gz": {Data: []byte("tar")}}
	none := fstest.MapFS{"requirements.yml": {Data: []byte("- name: a.one\n  version: 1.0.0\n")},
		"collections/.gitignore": {Data: []byte("*\n")}}
	tests := []struct {
		name    string
		fsys    fstest.MapFS
		version string
		want    string
	}{
		{"dot", testFS(), ".", "no name for a directory"},
		{"dots", testFS(), "..", "no name for a directory"},
		{"slash", testFS(), "v1/../x", "no name for a directory"},
		{"empty", testFS(), "", "no name for a directory"},
		{"no tarballs", none, "devel", "holds no collections: build it with tools/build"},
		{"another pin", bad, "devel", "differ from requirements.yml's pins: build it again"},
	}
	for _, tt := range tests {
		p, _, _ := testPlaybook(t, tt.fsys, tt.version)
		_, err := p.Plan(context.Background())
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: got %v, want %q", tt.name, err, tt.want)
		}
	}
}
