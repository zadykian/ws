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
