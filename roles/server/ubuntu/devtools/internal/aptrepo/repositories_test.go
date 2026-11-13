package aptrepo

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
)

const curl = "curl -fsSL --retry 5 --retry-delay 10 --retry-all-errors "

// armoredKey is a key as the keyserver serves it, with its Comment and Version lines.
func armoredKey(data []byte) string {
	return strings.Replace(armored(data), armorBegin+"\n", armorBegin+"\nVersion: x\nComment: y\n", 1)
}

type fixture struct {
	m     *effecttest.Machine
	repos func(string) []Repository
	keys  map[string]string // by repository
}

func newFixture(t *testing.T) fixture {
	m := effecttest.NewMachine()
	m.Files["/usr/lib/os-release"] = []byte("NAME=\"Ubuntu\"\nVERSION_CODENAME=resolute\n")
	f := fixture{m: m, keys: map[string]string{}}
	var list []Repository
	for _, name := range []string{"one", "two"} {
		e := newKey(t, false)
		url := "https://example.org/" + name + ".key"
		m.Results[curl+url] = effecttest.Result{Out: []byte(armoredKey(public(t, e)))}
		f.keys[name] = upper(e.PrimaryKey.Fingerprint)
		list = append(list, Repository{
			Name: name, Key: url, Fingerprints: []string{f.keys[name]}, URIs: "https://example.org/" + name,
			Suites: "*", Components: "main", Needed: true,
		})
	}
	f.repos = func(release string) []Repository {
		list[1].Suites = release
		return list
	}
	return f
}

func (f fixture) plan(_ *testing.T) ([]resource.Change, error) {
	return Repositories{M: f.m, List: f.repos, Replaced: replaced}.Plan(context.Background())
}

// replaced is the sources and keys that the fixture's run removes, as the role lists them.
func replaced(release string) []string {
	return []string{
		"/etc/apt/sources.list.d/nodesource.list",
		"/etc/apt/sources.list.d/dotnet-ubuntu-backports-" + release + ".sources",
		"/etc/apt/sources.list.d/github-cli.list",
		"/etc/apt/trusted.gpg.d/githubcli-archive-keyring.gpg",
	}
}

func apply(t *testing.T, changes []resource.Change) {
	for _, c := range changes {
		if err := c.Apply(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRepositoriesWriteKeysAndSources(t *testing.T) {
	f := newFixture(t)
	changes, err := f.plan(t)
	if err != nil || len(changes) != 4 {
		t.Fatalf("got %d changes, %v", len(changes), err)
	}
	apply(t, changes)
	key := string(f.m.Files["/etc/apt/keyrings/one.asc"])
	if !strings.Contains(key, armorBegin) || strings.Contains(key, "Comment:") || strings.Contains(key, "Version:") {
		t.Errorf("the key was written as %q", key)
	}
	want := "# Written by github.com/zadykian/ws, whose next run replaces any change made here.\n" +
		"Types: deb\nURIs: https://example.org/two\nSuites: resolute\nComponents: main\n" +
		"Signed-By: /etc/apt/keyrings/two.asc\n"
	if got := string(f.m.Files["/etc/apt/sources.list.d/two.sources"]); got != want {
		t.Errorf("the sources file is %q, want %q", got, want)
	}
	if changes, err := f.plan(t); err != nil || len(changes) != 0 {
		t.Errorf("a second plan has %v, %v", changes, err)
	}
}
