package server

import (
	"context"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/step"
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

func planStrings(t *testing.T, p Playbook) []string {
	pl, err := p.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range pl.Changes {
		got = append(got, c.String())
	}
	return got
}

func TestPlaybookPlan(t *testing.T) {
	t.Run("a new machine", func(t *testing.T) {
		p, _, _ := testPlaybook(t, testFS(), "v0.1.0")
		got := strings.Join(planStrings(t, p), "|")
		want := "write /etc/ws/host_vars.yml with no values|unpack the playbook into /var/lib/ws/v0.1.0"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
	t.Run("the same digest", func(t *testing.T) {
		p, m, digest := testPlaybook(t, testFS(), "v0.1.0")
		m.Dirs["/etc/ws"], m.Dirs[testLib] = 0o755, 0o755
		m.Files[testVars] = []byte("---\n")
		m.Files[testLib+"/v0.1.0/.unpacked"] = []byte(digest)
		if got := planStrings(t, p); len(got) != 0 {
			t.Errorf("got %v, want none", got)
		}
	})
	t.Run("another digest", func(t *testing.T) {
		p, m, _ := testPlaybook(t, testFS(), "v0.1.0")
		m.Dirs["/etc/ws"], m.Dirs[testLib] = 0o755, 0o755
		m.Files[testVars] = []byte("---\n")
		m.Files[testLib+"/v0.1.0/.unpacked"] = []byte("old")
		got := planStrings(t, p)
		if len(got) != 1 || !strings.HasPrefix(got[0], "unpack ") {
			t.Errorf("got %v, want the unpacking", got)
		}
	})
	t.Run("leftovers", func(t *testing.T) {
		p, m, digest := testPlaybook(t, testFS(), "devel")
		m.Dirs["/etc/ws"], m.Dirs[testLib] = 0o755, 0o755
		m.Files[testVars] = []byte("---\n")
		m.Files[testLib+"/devel/.unpacked"] = []byte(digest)
		m.Files[testLib+"/v0.0.9/.unpacked"] = []byte("x")
		m.Dirs[testLib+"/devel"], m.Dirs[testLib+"/v0.0.9"] = 0o755, 0o755
		m.Files[testLib+"/stray"] = nil
		got := planStrings(t, p)
		want := []string{"remove /var/lib/ws/stray, /var/lib/ws/v0.0.9"}
		if !slices.Equal(got, want) {
			t.Errorf("got %v, want the removal of the stray file and the old version", got)
		}
	})
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

func TestPlaybookApply(t *testing.T) {
	p, m, digest := testPlaybook(t, testFS(), "v0.1.0")
	d, n := testLib+"/v0.1.0", testLib+"/v0.1.0.new"
	for _, cmd := range []string{
		"rm -rf " + n,
		"env LC_ALL=C.UTF-8 ANSIBLE_COLLECTIONS_PATH=" + n + "/collections ansible-galaxy collection install --offline -p " +
			n + "/collections " + n + "/collections/a-one-1.0.0.tar.gz",
		"ln -s " + testVars + " " + n + "/host_vars/localhost.yml",
		"rm -rf " + d, "mv " + n + " " + d, "rm -rf " + testLib + "/old",
	} {
		m.Results[cmd] = effecttest.Result{}
	}
	pl := planOf(t, p, "old")
	if err := p.Apply(context.Background(), pl); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range m.Calls {
		if !strings.HasPrefix(c, "stat ") && !strings.HasPrefix(c, "read") {
			got = append(got, c)
		}
	}
	joined := strings.Join(got, "\n")
	for _, w := range []string{
		"mkdir /etc/ws 755\nwrite /etc/ws/host_vars.yml 600\nmkdir /var/lib/ws 755\nrun rm -rf " + n + "\nmkdir " + n + " 755",
		"write " + n + "/collections/a-one-1.0.0.tar.gz 644",
		"write " + n + "/roles/base/tasks/main.yml 644",
		"mkdir " + n + "/host_vars 755\nrun ln -s " + testVars + " " + n + "/host_vars/localhost.yml\nwrite " + n + "/.unpacked 644\nrun rm -rf " + d + "\nrun mv " + n + " " + d + "\nrun rm -rf " + testLib + "/old",
	} {
		if !strings.Contains(joined, w) {
			t.Errorf("calls lack %q:\n%s", w, joined)
		}
	}
	if string(m.Files[n+"/.unpacked"]) != digest {
		t.Errorf(".unpacked is %q, want the digest", m.Files[n+"/.unpacked"])
	}
}

// planOf returns the plan of a new machine with one leftover version.
func planOf(t *testing.T, p Playbook, leftover string) (pl step.Plan) {
	digest, err := Digest(p.FS)
	if err != nil {
		t.Fatal(err)
	}
	s := state{varsMissing: true, varsDirMissing: true, libMissing: true, entries: []string{leftover}}
	pl.Changes = changes(p.Lib, p.Vars, p.Version, digest, s)
	return pl
}
