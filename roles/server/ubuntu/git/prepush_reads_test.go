package git_test

import (
	"strings"
	"testing"
)

// These tests hold what the check reads of a commit against the settings and the text of a
// repository, which its tests of the rules in prepush_test.go leave out.

func TestPrePushRefusesACommitWhoseTextHoldsAControlCharacter(t *testing.T) {
	r := newRepo(t)
	r.good("base")
	for name, c := range map[string]struct{ committer, message string }{
		"an email":  {"a\x1fb@example.test", "x" + coAuthor},
		"a trailer": {wsMail, "x\n\nCo-authored-by: a\x1fb <c@example.test>"},
	} {
		sha := r.commit("", c.committer, wsMail, c.message)
		if err := r.check(line("refs/heads/main", sha)); err == nil || !strings.Contains(err.Error(), sha[:12]) {
			t.Errorf("%s: a commit that cannot be read passed or went unnamed: %v", name, err)
		}
		r.git("reset", "-q", "--hard", "HEAD~1")
	}
}

func TestPrePushNamesACommitWhereGitShowsSignatures(t *testing.T) {
	r := newRepo(t)
	r.git("config", "log.showSignature", "true")
	r.good("base")
	carried := r.commit(r.macKey, macMail, macMail, "carried")
	r.refused(r.check(line("refs/heads/main", carried)), carried, "signature is from a key")
}

func TestPrePushReadsTheObjectsAPushSends(t *testing.T) {
	r := newRepo(t)
	good := r.good("good")
	bad := r.commit("", wsMail, wsMail, "unsigned")
	r.git("replace", bad, good)
	r.refused(r.check(line("refs/heads/main", bad)), bad, "signature is missing")
}

// pushByURL sends main to the bare repository by its path, so that no remote-tracking ref holds
// its commits, as in a bare clone.
func (r repo) pushByURL() { r.git("push", "-q", r.remote, "main") }

func TestPrePushLeavesOutWhatTheRemoteHoldsAtTheRef(t *testing.T) {
	r := newRepo(t)
	old := r.commit("", wsMail, wsMail, "old, unsigned")
	r.pushByURL()
	tip := r.good("new")
	if err := r.check("refs/heads/main " + tip + " refs/heads/main " + old); err != nil {
		t.Errorf("a push of one credited commit onto the remote's branch: %v", err)
	}
	// Without the remote's id the whole history is walked.
	r.refused(r.check(line("refs/heads/main", tip)), old, "signature is missing")
}

func TestPrePushChecksAllOfARefNewToTheRemote(t *testing.T) {
	r := newRepo(t)
	old := r.commit("", wsMail, wsMail, "old, unsigned")
	r.pushByURL()
	tip := r.good("new")
	r.refused(r.check(line("refs/heads/other", tip)), old, "signature is missing")
}

func TestPrePushIgnoresAnObjectIdTheRepositoryLacks(t *testing.T) {
	r := newRepo(t)
	old := r.commit("", wsMail, wsMail, "old, unsigned")
	tip := r.good("new")
	r.refused(r.check("refs/heads/main "+tip+" refs/heads/main "+strings.Repeat("ab", 20)), old, "signature is missing")
}

func TestPrePushReadsALocalRefThatHoldsSpaces(t *testing.T) {
	r := newRepo(t)
	bad := r.commit("", wsMail, wsMail, "unsigned")
	in := "HEAD@{2 hours ago} " + bad + " refs/heads/x " + zeros
	r.refused(r.check(in), bad, "signature is missing")
	good := r.good("good")
	if err := r.check("HEAD@{2 hours ago} " + good + " refs/heads/x " + bad); err != nil {
		t.Errorf("a source with spaces onto the remote's commit: %v", err)
	}
}
