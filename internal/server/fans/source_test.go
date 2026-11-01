package fans

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

const sourceURL = "https://github.com/Fred78290/nct6687d/archive/" + nct6687Commit + ".tar.gz"

func TestSourceDownloadsUnpacksAndRemovesTheTarball(t *testing.T) {
	m := effecttest.NewMachine()
	body := []byte("tarball")
	m.Pages[sourceURL] = body
	var unpacked string
	s := source{m: m, sum: fmt.Sprintf("%x", sha256.Sum256(body)), unpack: func(archive, dir string, strip int) error {
		unpacked = fmt.Sprintf("%s %s %d", archive, dir, strip)
		return nil
	}}
	changes, err := s.Plan(t.Context())
	if err != nil || len(changes) != 2 {
		t.Fatalf("Plan = %d changes, %v; want the download and the unpack", len(changes), err)
	}
	for _, c := range changes {
		if err := c.Apply(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	if want := tarball + " " + sourceDir + " 1"; unpacked != want {
		t.Errorf("unpacked %q, want %q", unpacked, want)
	}
	if _, ok := m.Files[tarball]; ok {
		t.Error("the tarball stays")
	}
	if !strings.Contains(changes[1].String(), sourceDir) {
		t.Errorf("unpack change = %q", changes[1])
	}
}

func TestSourceInPlaceChangesNothing(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[sourceDir+"/nct6687.c"] = []byte("c")
	if changes, err := (source{m: m}).Plan(t.Context()); err != nil || len(changes) != 0 {
		t.Errorf("Plan = %v, %v; want none", changes, err)
	}
}

func TestSourceRefusesATarballOfAnotherSum(t *testing.T) {
	m := effecttest.NewMachine()
	m.Pages[sourceURL] = []byte("other")
	changes, err := source{m: m, sum: strings.Repeat("0", 64)}.Plan(t.Context())
	if err != nil || len(changes) != 2 || changes[0].Apply(t.Context()) == nil {
		t.Errorf("Plan = %v, %v; want a download that fails", changes, err)
	}
}
