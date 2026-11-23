package git_test

import (
	"strings"
	"testing"
)

// A line of three dashes in the body is no start of a patch: the trailer stays in the last
// paragraph, where the push check's %(trailers) reads it.
func TestGitHookKeepsTheTrailerBelowALineOfDashes(t *testing.T) {
	skipBelowHooks(t)
	r := newGitRepo(t)
	r.git("commit", "-q", "--allow-empty", "-m", "fix: a thing\n\nWhy it changes.\n\n---\n\nNotes below a rule.")
	want := strings.TrimPrefix(trailer(), "Co-authored-by: ")
	if got := strings.TrimSpace(r.git("log", "-1", "--format=%(trailers:key=Co-authored-by,valueonly)")); got != want {
		t.Errorf("trailers of a message with a rule = %q, want %q", got, want)
	}
	if n := r.count(); n != 1 {
		t.Errorf("the message holds %d trailers, want 1", n)
	}
}
