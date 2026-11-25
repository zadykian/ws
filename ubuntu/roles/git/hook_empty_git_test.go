package git_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Where no editor opens, git gives the hook GIT_EDITOR=: and cleans the message by whitespace, so
// a line that starts with # is text, as in an issue's number.
func TestGitHookKeepsACommentLookingMessageWhereNoEditorOpens(t *testing.T) {
	skipBelowHooks(t)
	r := newGitRepo(t)
	r.git("commit", "-q", "--allow-empty", "-m", "#42 fix the fan curve")
	if n := r.count(); n != 1 {
		t.Errorf("-m with a # line holds %d trailers, want 1", n)
	}
	file := filepath.Join(r.dir, "msg")
	if err := os.WriteFile(file, []byte("#43 more\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r.git("commit", "-q", "--allow-empty", "-F", file)
	if n := r.count(); n != 1 {
		t.Errorf("-F with a # line holds %d trailers, want 1", n)
	}
	r.git("commit", "-q", "--allow-empty", "--amend", "--no-edit")
	if n := r.count(); n != 1 {
		t.Errorf("an amend with no edit holds %d trailers, want 1", n)
	}
}

// git cuts the scissors section by its comment string, and the hook does as git does.
func TestGitHookLeavesAnEmptyMessageToAbortUnderAnyCommentChar(t *testing.T) {
	skipBelowHooks(t)
	r := newGitRepo(t)
	r.git("commit", "-q", "--allow-empty", "-m", "one")
	head := strings.TrimSpace(r.git("rev-parse", "HEAD"))
	if err := os.WriteFile(filepath.Join(r.dir, "new"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	r.git("add", "new")
	for _, char := range []string{"#", ";", "@"} {
		r.git("config", "core.commentChar", char)
		if out, err := r.run([]string{"GIT_EDITOR=true"}, "commit", "-q", "-v"); err == nil {
			t.Errorf("commentChar %q: an empty message with -v went through, want git to abort:\n%s", char, out)
		}
		if got := strings.TrimSpace(r.git("rev-parse", "HEAD")); got != head {
			t.Errorf("commentChar %q: HEAD moved from %s to %s", char, head, got)
			r.git("reset", "-q", "--soft", head)
		}
	}
}
