package server

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"slices"
	"testing"
	"testing/fstest"
)

func TestPins(t *testing.T) {
	tests := []struct {
		name, in string
		want     []string
		fail     bool
	}{
		{"quoted", "collections:\n  - name: \"b.two\"\n    version: '2.0.0'\n  - name: a.one\n    version: 1.0.0\n",
			[]string{"a-one-1.0.0.tar.gz", "b-two-2.0.0.tar.gz"}, false},
		{"comments and underscores", "# c\n- name: x.y_z_v1\n  # why\n  version: 1.1.5\n",
			[]string{"x-y_z_v1-1.1.5.tar.gz"}, false},
		{"none", "collections: []\n", nil, false},
		{"no version", "- name: a.one\n", nil, true},
		{"no version before the next", "- name: a.one\n- name: b.two\n  version: 1\n", nil, true},
		{"version alone", "  version: 1\n", nil, true},
	}
	for _, tt := range tests {
		got, err := Pins([]byte(tt.in))
		if (err != nil) != tt.fail || !slices.Equal(got, tt.want) {
			t.Errorf("%s: got %v, %v; want %v, fail %v", tt.name, got, err, tt.want, tt.fail)
		}
	}
}

func TestPinsOfRequirements(t *testing.T) {
	data, err := os.ReadFile("../../../requirements.yml")
	if err != nil {
		t.Fatal(err)
	}
	got, err := Pins(data)
	want := []string{
		"community-docker-5.4.0.tar.gz",
		"community-general-13.4.0.tar.gz",
		"community-library_inventory_filtering_v1-1.1.5.tar.gz",
	}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("got %v, %v; want %v", got, err, want)
	}
}

func TestDigest(t *testing.T) {
	fsys := fstest.MapFS{
		"b.txt":     {Data: []byte("two")},
		"a/c.txt":   {Data: []byte("one")},
		"a/empty":   {Data: nil},
		"a/dir/. x": {Data: []byte("x")},
	}
	got, err := Digest(fsys)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.New()
	sum.Write([]byte("a/c.txt\x003\x00one" + "a/dir/. x\x001\x00x" + "a/empty\x000\x00" + "b.txt\x003\x00two"))
	if want := hex.EncodeToString(sum.Sum(nil)); got != want {
		t.Errorf("got %s, want %s", got, want)
	}
	again, _ := Digest(fsys)
	fsys["b.txt"] = &fstest.MapFile{Data: []byte("tw0")}
	changed, _ := Digest(fsys)
	if again != got || changed == got {
		t.Error("the digest is not a function of the bytes")
	}
}

func TestDigestOfNothing(t *testing.T) {
	got, err := Digest(fstest.MapFS{})
	if want := hex.EncodeToString(sha256.New().Sum(nil)); err != nil || got != want {
		t.Errorf("got %s, %v", got, err)
	}
}

func TestStages(t *testing.T) {
	if packagesStage == playbookStage || packagesStage == "" {
		t.Error("the stages must differ")
	}
}
