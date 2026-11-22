package git_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/ubuntu/git"
)

const (
	wsMail   = "su@zadykian.dev"
	macMail  = "me@zadykian.dev"
	coAuthor = "\n\nCo-authored-by: zadykian <me@zadykian.dev>"
	zeros    = "0000000000000000000000000000000000000000"
)

// pushMachine runs git for real in the test's repository, with a config of the test's own, where
// the fake answers the rest.
type pushMachine struct {
	*effecttest.Machine
	repo
}

func (m pushMachine) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // the check names git and its arguments
	cmd.Dir, cmd.Env = m.dir, m.env
	return cmd.Output()
}

// repo is a repository whose signer is a key of its own, trusted for ws's email alone, with a bare
// repository to push to. Neither the host's config nor its environment reaches git.
type repo struct {
	t              *testing.T
	dir, remote    string
	env            []string
	wsKey, macKey  string
	publicKeyBytes string
}

func newRepo(t *testing.T) repo {
	home := t.TempDir()
	r := repo{t: t, dir: filepath.Join(home, "work"), remote: filepath.Join(home, "remote.git"),
		wsKey: filepath.Join(home, "ws"), macKey: filepath.Join(home, "mac")}
	for _, key := range []string{r.wsKey, r.macKey} {
		if out, err := exec.CommandContext(t.Context(), "ssh-keygen", "-q", "-t", "ed25519", "-N", "", "-f", key).CombinedOutput(); err != nil { //nolint:gosec // the test's own key
			t.Skipf("ssh-keygen: %v: %s", err, out)
		}
	}
	pub, err := os.ReadFile(r.wsKey + ".pub")
	if err != nil {
		t.Fatal(err)
	}
	r.publicKeyBytes = strings.Fields(string(pub))[1]
	signers := filepath.Join(home, "allowed_signers")
	global := filepath.Join(home, "gitconfig")
	for path, text := range map[string]string{
		signers: wsMail + " " + string(pub),
		global: "[gpg]\n\tformat = ssh\n[gpg \"ssh\"]\n\tallowedSignersFile = " + signers +
			"\n[user]\n\tname = t\n\temail = " + wsMail + "\n[init]\n\tdefaultBranch = main\n[tag]\n\tgpgsign = false\n",
	} {
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	r.env = []string{"HOME=" + home, "PATH=" + os.Getenv("PATH"), "GIT_CONFIG_GLOBAL=" + global,
		"GIT_CONFIG_SYSTEM=/dev/null", "GIT_CONFIG_NOSYSTEM=1"}
	for _, d := range []string{r.dir, r.remote} {
		if err := os.Mkdir(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	r.git("-C", r.remote, "init", "-q", "--bare")
	r.git("init", "-q")
	return r
}

// git runs git in the repository, and returns its output.
func (r repo) git(args ...string) string {
	cmd := exec.CommandContext(r.t.Context(), "git", args...) //nolint:gosec // the test's own git arguments
	cmd.Dir, cmd.Env = r.dir, r.env
	out, err := cmd.CombinedOutput()
	if err != nil {
		r.t.Fatalf("git %q: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// commit makes an empty commit signed by key, or unsigned for none, as committer and author, and
// returns its hash.
func (r repo) commit(key, committer, author, message string) string {
	args := []string{"-c", "user.name=t", "-c", "user.email=" + committer, "-c", "user.signingkey=" + key,
		"-c", "commit.gpgsign=" + map[bool]string{true: "true", false: "false"}[key != ""],
		"commit", "-q", "--allow-empty", "--author=t <" + author + ">", "-m", message}
	r.git(args...)
	return r.git("rev-parse", "HEAD")
}

// good is a commit of ws as the hook leaves it: signed, and credited.
func (r repo) good(subject string) string {
	return r.commit(r.wsKey, wsMail, wsMail, subject+coAuthor)
}

// push sends main to the bare repository, as a push that git does, so that a remote-tracking
// ref holds its commits.
func (r repo) push() {
	r.git("remote", "add", "origin", r.remote)
	r.git("push", "-q", "origin", "main")
}

// check runs ws's check on the pushed refs, one line each as git writes them.
func (r repo) check(refs ...string) error {
	var in strings.Builder
	for _, ref := range refs {
		in.WriteString(ref + "\n")
	}
	return git.PrePush(r.t.Context(), pushMachine{effecttest.NewMachine(), r}, strings.NewReader(in.String()))
}

// line is a pushed ref's line, to a branch that is new on the remote.
func line(ref, sha string) string { return ref + " " + sha + " " + ref + " " + zeros }

// refused fails unless err names the commit and the rule, and no key.
func (r repo) refused(err error, sha string, rules ...string) {
	r.t.Helper()
	if err == nil {
		r.t.Fatalf("the push of %s passed, want a refusal", sha[:12])
	}
	for _, want := range append([]string{sha[:12]}, rules...) {
		if !strings.Contains(err.Error(), want) {
			r.t.Errorf("the refusal lacks %q: %v", want, err)
		}
	}
	for _, secret := range []string{r.publicKeyBytes, r.wsKey, r.macKey} {
		if strings.Contains(err.Error(), secret) {
			r.t.Errorf("the refusal holds a key or a key's path: %v", err)
		}
	}
}

func TestPrePushPassesSignedCommitsWithTheTrailer(t *testing.T) {
	r := newRepo(t)
	r.good("one")
	tip := r.good("two")
	if err := r.check(line("refs/heads/main", tip)); err != nil {
		t.Errorf("a signed and credited branch: %v", err)
	}
}

func TestPrePushRefusesEachRule(t *testing.T) {
	r := newRepo(t)
	r.good("base")
	for _, c := range []struct{ name, key, committer, message, rule string }{
		{"unsigned", "", wsMail, "x" + coAuthor, "signature is missing"},
		{"no trailer", r.wsKey, wsMail, "x", "trailer Co-authored-by: zadykian <" + macMail + "> is missing"},
		{"another committer", r.wsKey, "other@example.test", "x" + coAuthor, "signer " + wsMail + " is not the committer other@example.test"},
		{"unknown key", r.macKey, wsMail, "x" + coAuthor, "signature is from a key that /etc/ws/allowed_signers lacks"},
	} {
		sha := r.commit(c.key, c.committer, wsMail, c.message)
		r.refused(r.check(line("refs/heads/main", sha)), sha, c.rule)
		r.git("reset", "-q", "--hard", "HEAD~1")
	}
}

func TestPrePushListsEachBrokenRuleOfEachCommit(t *testing.T) {
	r := newRepo(t)
	a := r.commit("", wsMail, wsMail, "a")
	b := r.good("b")
	err := r.check(line("refs/heads/main", b))
	r.refused(err, a, "signature is missing", "trailer")
	if strings.Contains(err.Error(), b[:12]) {
		t.Errorf("the refusal names the good commit: %v", err)
	}
}

func TestPrePushLeavesADeletedRefAlone(t *testing.T) {
	r := newRepo(t)
	sha := r.commit("", wsMail, wsMail, "unsigned")
	if err := r.check("refs/heads/gone " + zeros + " refs/heads/gone " + sha); err != nil {
		t.Errorf("a deleted ref: %v", err)
	}
}

func TestPrePushChecksTagsByTheirCommits(t *testing.T) {
	r := newRepo(t)
	r.good("base")
	r.push()
	r.git("tag", "held")
	r.git("tag", "-a", "-m", "held", "held-annotated")
	bad := r.commit("", wsMail, wsMail, "unsigned")
	r.git("tag", "light")
	r.git("tag", "-a", "-m", "annotated", "annotated")
	tagObject := r.git("rev-parse", "annotated")
	for _, tag := range []string{"held", "held-annotated"} {
		if err := r.check(line("refs/tags/"+tag, r.git("rev-parse", tag))); err != nil {
			t.Errorf("tag %s of a pushed commit: %v", tag, err)
		}
	}
	r.refused(r.check(line("refs/tags/light", bad)), bad, "signature")
	r.refused(r.check(line("refs/tags/annotated", tagObject)), bad, "signature")
	tree := r.git("rev-parse", "HEAD^{tree}")
	r.git("tag", "tree-tag", tree)
	if err := r.check(line("refs/tags/tree-tag", tree)); err != nil {
		t.Errorf("a tag of a tree sends no commit: %v", err)
	}
}

func TestPrePushChecksAMergeCommit(t *testing.T) {
	r := newRepo(t)
	r.good("base")
	r.git("switch", "-q", "-c", "side")
	r.good("side")
	r.git("switch", "-q", "main")
	r.good("main")
	r.push()
	r.git("-c", "user.name=t", "-c", "user.email="+wsMail, "-c", "user.signingkey="+r.wsKey, "-c", "commit.gpgsign=true",
		"merge", "-q", "--no-ff", "-m", "merge", "side")
	merge := r.git("rev-parse", "HEAD")
	r.refused(r.check(line("refs/heads/main", merge)), merge, "trailer")
	r.git("-c", "user.name=t", "-c", "user.email="+wsMail, "-c", "user.signingkey="+r.wsKey, "-c", "commit.gpgsign=true",
		"commit", "-q", "--amend", "-m", "merge"+coAuthor)
	if err := r.check(line("refs/heads/main", r.git("rev-parse", "HEAD"))); err != nil {
		t.Errorf("a credited, signed merge and the side's commit: %v", err)
	}
}

func TestPrePushChecksTheCommitsAMergeBringsIn(t *testing.T) {
	r := newRepo(t)
	r.good("base")
	r.push()
	r.git("switch", "-q", "-c", "side")
	side := r.commit("", wsMail, wsMail, "unsigned"+coAuthor)
	r.git("switch", "-q", "main")
	r.good("main")
	r.git("-c", "user.name=t", "-c", "user.email="+wsMail, "-c", "user.signingkey="+r.wsKey, "-c", "commit.gpgsign=true",
		"merge", "-q", "--no-ff", "-m", "merge"+coAuthor, "side")
	r.refused(r.check(line("refs/heads/main", r.git("rev-parse", "HEAD"))), side, "signature is missing")
}

func TestPrePushTakesTheMacsCommitsAsWsSignsThem(t *testing.T) {
	r := newRepo(t)
	r.good("base")
	r.push()
	// A commit of the Mac's, which ws rebased: its author is the maintainer, its committer and its
	// signer ws, and the hook never saw its message.
	rebased := r.commit(r.wsKey, wsMail, macMail, "from the Mac")
	if err := r.check(line("refs/heads/main", rebased)); err != nil {
		t.Errorf("a Mac commit that ws signed: %v", err)
	}
	// One that ws only carries keeps the Mac's signature, which ws cannot check.
	carried := r.commit(r.macKey, macMail, macMail, "carried")
	r.refused(r.check(line("refs/heads/main", carried)), carried, "signature is from a key that /etc/ws/allowed_signers lacks")
}

func TestPrePushChecksAllOfARefWithoutRemoteTrackingRefs(t *testing.T) {
	r := newRepo(t)
	old := r.commit("", wsMail, wsMail, "old")
	tip := r.good("new")
	r.refused(r.check(line("refs/heads/main", tip)), old, "signature is missing")
}

func TestPrePushPassesAnEmptyRange(t *testing.T) {
	r := newRepo(t)
	held := r.commit("", wsMail, wsMail, "held by the remote")
	r.push()
	r.git("branch", "again")
	for _, ref := range []string{"main", "again"} {
		if err := r.check(line("refs/heads/"+ref, held)); err != nil {
			t.Errorf("%s holds no new commit: %v", ref, err)
		}
	}
	if err := r.check(); err != nil {
		t.Errorf("no ref: %v", err)
	}
}

func TestPrePushRefusesALineItCannotRead(t *testing.T) {
	r := newRepo(t)
	for _, in := range []string{"refs/heads/main", "refs/heads/main abc refs/heads/main " + zeros, "a --all b c"} {
		if err := r.check(in); err == nil {
			t.Errorf("line %q passed", in)
		}
	}
}
