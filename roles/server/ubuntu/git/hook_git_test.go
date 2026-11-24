package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/gitcfg"
)

// trailer is the line that each commit on ws holds once: the Mac's identity, which the maintainer
// commits as.
func trailer() string {
	id := gitcfg.Mac()
	return "Co-authored-by: " + id.Name + " <" + id.Email + ">"
}

// gitRepo is a repository of the test's own, read by a git whose config is the rendered file alone.
type gitRepo struct {
	t   *testing.T
	dir string
	env []string
}

// run is git with the extra environment variables beside the repository's own.
func (r gitRepo) run(extra []string, args ...string) (string, error) {
	cmd := exec.CommandContext(r.t.Context(), "git", append([]string{"-C", r.dir}, args...)...) //nolint:gosec // the test's own git arguments
	cmd.Env = append(append([]string{}, r.env...), extra...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (r gitRepo) git(args ...string) string {
	out, err := r.run(nil, args...)
	if err != nil {
		r.t.Fatalf("git %q: %v\n%s", args, err, out)
	}
	return out
}

// count is how many times the trailer stands in the message of the last commit.
func (r gitRepo) count() int {
	return strings.Count(r.git("log", "-1", "--format=%B"), trailer()+"\n")
}

// newGitRepo renders ws's settings into a file, and has a global config that includes it, with
// signing off, which needs a key. Neither the host's config nor its environment reaches git. The
// test's binary stands for ws in the hooks (TestMain).
func newGitRepo(t *testing.T) gitRepo {
	m, e := machine(t)
	if err := write(t, e, units(t, e, "file/git-config")); err != nil {
		t.Fatal(err)
	}
	ownBinary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	ws := filepath.Join(home, "ws.gitconfig")
	global := filepath.Join(home, "gitconfig")
	for path, text := range map[string]string{
		ws:     strings.ReplaceAll(string(m.Files["/etc/ws/gitconfig"]), self, ownBinary),
		global: "[include]\n\tpath = " + ws + "\n[commit]\n\tgpgsign = false\n[tag]\n\tgpgsign = false\n[init]\n\tdefaultBranch = main\n",
	} {
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	r := gitRepo{t: t, dir: filepath.Join(home, "repo"), env: []string{
		"HOME=" + home, "PATH=" + os.Getenv("PATH"), "GIT_CONFIG_GLOBAL=" + global,
		"GIT_CONFIG_SYSTEM=/dev/null", "GIT_CONFIG_NOSYSTEM=1", asWs + "=1",
	}}
	if err := os.Mkdir(r.dir, 0o700); err != nil {
		t.Fatal(err)
	}
	r.git("init", "-q")
	return r
}

// skipBelowHooks skips a test that needs the hooks of git's config, which came in git 2.54.
func skipBelowHooks(t *testing.T) {
	out, err := exec.CommandContext(t.Context(), "git", "version").Output()
	if err != nil {
		t.Skip("no git")
	}
	var major, minor int
	if f := strings.FieldsFunc(strings.TrimPrefix(string(out), "git version "), func(r rune) bool { return r == '.' || r == ' ' }); len(f) >= 2 {
		major, _ = strconv.Atoi(f[0])
		minor, _ = strconv.Atoi(f[1])
	}
	if major < 2 || major == 2 && minor < 54 {
		t.Skipf("%s: hooks in git's config need 2.54", strings.TrimSpace(string(out)))
	}
}

func TestGitHookAddsTheTrailerOnce(t *testing.T) {
	skipBelowHooks(t)
	r := newGitRepo(t)
	r.git("commit", "-q", "--allow-empty", "-m", "one")
	if n := r.count(); n != 1 {
		t.Errorf("a commit with -m holds %d trailers, want 1", n)
	}
	r.git("commit", "-q", "--allow-empty", "--amend", "--no-edit")
	if n := r.count(); n != 1 {
		t.Errorf("an amend of the message holds %d trailers, want 1", n)
	}
	r.git("commit", "-q", "--allow-empty", "--amend", "-m", "two\n\nbody")
	if n := r.count(); n != 1 {
		t.Errorf("an amend with a new message holds %d trailers, want 1", n)
	}
	r.git("commit", "-q", "--allow-empty", "-m", "three\n\n"+trailer())
	if n := r.count(); n != 1 {
		t.Errorf("a message that holds the trailer already has %d, want 1", n)
	}
}

func TestGitHookAddsTheTrailerToAMerge(t *testing.T) {
	skipBelowHooks(t)
	r := newGitRepo(t)
	r.git("commit", "-q", "--allow-empty", "-m", "base")
	r.git("switch", "-q", "-c", "side")
	r.git("commit", "-q", "--allow-empty", "-m", "side")
	r.git("switch", "-q", "main")
	r.git("commit", "-q", "--allow-empty", "-m", "main")
	r.git("merge", "-q", "--no-ff", "-m", "merge side", "side")
	if n := r.count(); n != 1 {
		t.Errorf("a merge commit holds %d trailers, want 1", n)
	}
}

// The hook of the repository runs too, and after ws's. Its hooksPath is the one of ws's own
// .githooks.
func TestGitHookRunsBesideARepositorysOwnHooksPath(t *testing.T) {
	skipBelowHooks(t)
	r := newGitRepo(t)
	hooks := filepath.Join(r.dir, ".githooks")
	ran := filepath.Join(r.dir, "own-hook-ran")
	if err := os.MkdirAll(hooks, 0o700); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\ngrep -c '^" + trailer() + "$' \"$1\" >" + ran + "\n"
	if err := os.WriteFile(filepath.Join(hooks, "commit-msg"), []byte(script), 0o755); err != nil { //nolint:gosec // a test's hook runs
		t.Fatal(err)
	}
	r.git("config", "core.hooksPath", ".githooks")
	r.git("commit", "-q", "--allow-empty", "-m", "one")
	if n := r.count(); n != 1 {
		t.Errorf("a commit under the repository's hooksPath holds %d trailers, want 1", n)
	}
	got, err := os.ReadFile(ran) //nolint:gosec // the marker of the test's own hook
	if err != nil || strings.TrimSpace(string(got)) != "1" {
		t.Errorf("the repository's own hook saw %q, %v; want it to run after ws's, with one trailer", got, err)
	}
}

// An empty message, once its comments and its scissors section are cut, is git's way to abort a
// commit. The hook leaves it to git, as a trailer alone would commit.
func TestGitHookLeavesAnEmptyMessageToAbort(t *testing.T) {
	skipBelowHooks(t)
	r := newGitRepo(t)
	r.git("commit", "-q", "--allow-empty", "-m", "one")
	head := r.git("rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(r.dir, "new"), []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}
	r.git("add", "new")
	for _, c := range []struct {
		name, editor string
		args         []string
	}{
		{"-m with no text", "true", []string{"commit", "-q", "-m", ""}},
		{"a template of comments", "true", []string{"commit", "-q"}},
		{"a template and a diff", "true", []string{"commit", "-q", "-v"}},
		{"an amend emptied by editing", "truncate -s0", []string{"commit", "-q", "--amend"}},
	} {
		if out, err := r.run([]string{"GIT_EDITOR=" + c.editor}, c.args...); err == nil {
			t.Errorf("%s: the commit went through, want git to abort:\n%s", c.name, out)
		}
		if got := r.git("rev-parse", "HEAD"); got != head {
			t.Errorf("%s: HEAD moved from %s to %s", c.name, head, got)
			r.git("reset", "-q", "--soft", head)
		}
	}
	if n := r.count(); n != 1 {
		t.Errorf("the message of HEAD holds %d trailers, want its own 1", n)
	}
}

// git cherry-pick and git rebase run no commit-msg hook: a commit that they write keeps its
// message. A reword of an interactive rebase does run it, as it edits the message.
func TestGitHookDoesNotRunOnACherryPickOrARebase(t *testing.T) {
	skipBelowHooks(t)
	r := newGitRepo(t)
	change := func(name string) {
		if err := os.WriteFile(filepath.Join(r.dir, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
		r.git("add", name)
	}
	change("base")
	r.git("commit", "-q", "-m", "base")
	for _, name := range []string{"side", "other"} {
		r.git("switch", "-q", "-c", name, "main")
		change(name)
		r.git("commit", "-q", "--no-verify", "-m", name)
	}
	r.git("switch", "-q", "main")
	change("main")
	r.git("commit", "-q", "-m", "main")
	r.git("cherry-pick", "side")
	if n := r.count(); n != 0 {
		t.Errorf("a cherry-picked commit holds %d trailers, want its message as it was", n)
	}
	r.git("switch", "-q", "other")
	before := strings.TrimSpace(r.git("rev-parse", "HEAD"))
	r.git("rebase", "-q", "main")
	if after := strings.TrimSpace(r.git("rev-parse", "HEAD")); after == before {
		t.Fatal("the rebase left its commit as it was, so it proves nothing")
	}
	if n := r.count(); n != 0 {
		t.Errorf("a rebased commit holds %d trailers, want its message as it was", n)
	}
	if out, err := r.run([]string{"GIT_SEQUENCE_EDITOR=sed -i 1s/^pick/reword/", "GIT_EDITOR=true"}, "rebase", "-q", "-i", "main"); err != nil {
		t.Fatalf("git rebase -i: %v\n%s", err, out)
	}
	if n := r.count(); n != 1 {
		t.Errorf("a reworded commit holds %d trailers, want 1", n)
	}
}
