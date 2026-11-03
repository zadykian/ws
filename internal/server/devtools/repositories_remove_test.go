package devtools

import (
	"strings"
	"testing"
)

func TestRepositoriesRemoveWhatTheyReplaceFirst(t *testing.T) {
	f := newFixture(t)
	f.m.Files["/etc/apt/sources.list.d/nodesource.list"] = []byte("deb x")
	f.m.Files["/etc/apt/sources.list.d/dotnet-ubuntu-backports-resolute.sources"] = []byte("x")
	changes, err := f.plan(t)
	if err != nil || len(changes) != 6 {
		t.Fatalf("got %d changes, %v", len(changes), err)
	}
	for i, want := range []string{"nodesource.list", "dotnet-ubuntu-backports-resolute.sources"} {
		if !strings.HasPrefix(changes[i].String(), "remove ") || !strings.HasSuffix(changes[i].String(), want) {
			t.Errorf("change %d is %q, want a removal of %s", i, changes[i], want)
		}
	}
	apply(t, changes)
	if changes, err := f.plan(t); err != nil || len(changes) != 0 {
		t.Errorf("a second plan has %v, %v", changes, err)
	}
}

func TestRepositoriesRemoveThoseNoLongerNeeded(t *testing.T) {
	f := newFixture(t)
	inner := f.repos
	f.repos = func(release string) []Repository {
		list := inner(release)
		list[1].Needed = false
		return list
	}
	f.m.Files["/etc/apt/keyrings/two.asc"] = []byte("k")
	f.m.Files["/etc/apt/sources.list.d/two.sources"] = []byte("s")
	changes, err := f.plan(t)
	if err != nil || len(changes) != 4 {
		t.Fatalf("got %d changes, %v", len(changes), err)
	}
	apply(t, changes)
	if _, ok := f.m.Files["/etc/apt/keyrings/two.asc"]; ok {
		t.Error("the key stayed")
	}
	if changes, err := f.plan(t); err != nil || len(changes) != 0 {
		t.Errorf("a second plan has %v, %v", changes, err)
	}
}
