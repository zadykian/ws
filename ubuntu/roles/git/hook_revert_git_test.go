package git_test

import (
	"os"
	"path/filepath"
	"testing"
)

// git revert runs no commit-msg hook, so its commit has no trailer, until an amend runs the hook.
func TestGitHookDoesNotRunOnARevertAndAnAmendMendsIt(t *testing.T) {
	skipBelowHooks(t)
	r := newGitRepo(t)
	r.git("commit", "-q", "--allow-empty", "-m", "base")
	if err := os.WriteFile(filepath.Join(r.dir, "new"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	r.git("add", "new")
	r.git("commit", "-q", "-m", "add new")
	r.git("revert", "--no-edit", "HEAD")
	if n := r.count(); n != 0 {
		t.Errorf("a revert holds %d trailers, want its message as git wrote it", n)
	}
	r.git("commit", "-q", "--amend", "--no-edit")
	if n := r.count(); n != 1 {
		t.Errorf("an amend of a revert holds %d trailers, want 1", n)
	}
}
