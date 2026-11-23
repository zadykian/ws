package setup_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/local"
)

// elsewhere makes another repository, with a tag and a staged file, and names it to git as a hook
// or a rebase's exec does: by GIT_DIR, GIT_WORK_TREE and GIT_INDEX_FILE. It runs after upstream,
// which drops them. It returns a check that the repository's tags, HEAD and index are as they were.
func elsewhere(t *testing.T) func() {
	outer := t.TempDir()
	git(t, outer, "init", "-q")
	release(t, outer, "v0.0.1")
	if err := os.WriteFile(filepath.Join(outer, "staged"), []byte("the user's change"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, outer, "add", "staged")
	dotGit := filepath.Join(outer, ".git")
	head, index := content(t, filepath.Join(dotGit, "HEAD")), content(t, filepath.Join(dotGit, "index"))
	t.Setenv("GIT_DIR", dotGit)
	t.Setenv("GIT_WORK_TREE", outer)
	t.Setenv("GIT_INDEX_FILE", filepath.Join(dotGit, "index"))
	return func() {
		t.Helper()
		if got := git(t, outer, "tag", "--list"); got != "v0.0.1" {
			t.Errorf("the other repository's tags = %q", got)
		}
		if got := content(t, filepath.Join(dotGit, "HEAD")); got != head {
			t.Errorf("the other repository's HEAD = %q, was %q", got, head)
		}
		if content(t, filepath.Join(dotGit, "index")) != index {
			t.Error("the other repository's index changed")
		}
	}
}

func TestCheckoutLeavesTheRepositoryThatGitDirNamesAlone(t *testing.T) {
	c, _ := upstream(t, "v1.2.3")
	unchanged := elsewhere(t)
	if n := run(t, c); n != 1 {
		t.Fatalf("first run planned %d changes", n)
	}
	unchanged()
	if info, err := os.Stat(filepath.Join(c.Dir, ".git")); err != nil || !info.IsDir() {
		t.Fatalf("%s holds no .git of its own: %v", c.Dir, err)
	}
	if got := content(t, filepath.Join(c.Dir, "mise.toml")); got != "# v1.2.3\n" {
		t.Errorf("mise.toml = %q", got)
	}
	if n := run(t, c); n != 0 {
		t.Errorf("a plan after the run found %d changes", n)
	}
	unchanged()
}

func TestCheckoutAsUserFindsNoChangeAfterItsRunWithGitDirSet(t *testing.T) {
	c, _ := upstream(t, "v1.2.3")
	unchanged := elsewhere(t)
	// sudo drops the repository's variables, as its env_reset does, so that the reads alone see them.
	c.M = local.OSMachine{Sudo: []string{"env", "-u", "GIT_DIR", "-u", "GIT_WORK_TREE", "-u", "GIT_INDEX_FILE"}}
	c.AsUser = true
	if n := run(t, c); n != 1 {
		t.Fatalf("first run planned %d changes", n)
	}
	if n := run(t, c); n != 0 {
		t.Errorf("a plan after the run found %d changes", n)
	}
	unchanged()
}
