package fans

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const sourceURL = "https://github.com/Fred78290/nct6687d/archive/" + nct6687Commit + ".tar.gz"

func TestSourceTreeFetchesTheCommitsTarballAndStripsItsTopDirectory(t *testing.T) {
	m := effecttest.NewMachine()
	tree := sourceTree(m)
	changes, err := tree.Plan(t.Context())
	if err != nil || len(changes) != 3 {
		t.Fatalf("Plan = %d changes, %v; want the directory, the download and the install", len(changes), err)
	}
	if want := "download " + sourceURL + " to /var/cache/ws/" + nct6687Commit + ".tar.gz"; changes[1].String() != want {
		t.Errorf("change = %q, want %q", changes[1], want)
	}
	if want := "install " + sourceURL + " in " + sourceDir; changes[2].String() != want {
		t.Errorf("change = %q, want %q", changes[2], want)
	}
	if tree.Strip != 1 || tree.Source.SHA256["all"] != nct6687SHA256 {
		t.Errorf("strip %d, sums %v", tree.Strip, tree.Source.SHA256)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return c == "run dpkg --print-architecture" }) {
		t.Errorf("the source read the architecture: %q", m.Calls)
	}
}
