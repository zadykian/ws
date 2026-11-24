package setup_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/effect/local"
	"github.com/zadykian/ws/internal/setup"
)

// gitEnv is the environment of the git that a test starts: its own, with no config of the machine.
func gitEnv(home string) []string {
	return []string{"PATH=" + os.Getenv("PATH"), "HOME=" + home, "GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.org",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.org"}
}

// git runs git in dir with the test's own environment, and returns its output.
func git(t *testing.T, dir string, args ...string) string {
	cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...) //nolint:gosec // the arguments are this test's own
	cmd.Env = gitEnv(t.TempDir())
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// release commits mise.toml with the content of tag to the repository in dir, and tags it. The
// repository ignores mise.local.toml, as ws's does.
func release(t *testing.T, dir, tag string) {
	if err := os.WriteFile(filepath.Join(dir, "mise.toml"), []byte("# "+tag+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("/mise.local.toml\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "mise.toml", ".gitignore")
	git(t, dir, "commit", "-q", "-m", tag)
	git(t, dir, "tag", tag)
}

// upstream returns a repository with the releases of tags, and the tree's Checkout of the first.
func upstream(t *testing.T, tags ...string) (setup.Checkout, string) {
	// The test's environment loses each GIT_ variable, such as the GIT_DIR that a rebase's exec or a
	// hook exports, and names no config of the machine. A test that names a repository by GIT_DIR
	// sets it after this.
	for _, entry := range os.Environ() {
		if name, _, _ := strings.Cut(entry, "="); strings.HasPrefix(name, "GIT_") {
			t.Setenv(name, "")
			if err := os.Unsetenv(name); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	repo := t.TempDir()
	git(t, repo, "init", "-q")
	for _, tag := range tags {
		release(t, repo, tag)
	}
	dir := filepath.Join(t.TempDir(), "ws")
	return setup.Checkout{M: local.OSMachine{Sudo: []string{}}, Dir: dir, Repo: "file://" + repo, Tag: tags[0]}, repo
}

// run plans the checkout and applies its changes, and returns how many it planned.
func run(t *testing.T, c setup.Checkout) int {
	changes, err := c.Plan(t.Context())
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	for _, change := range changes {
		if err := change.Apply(t.Context()); err != nil {
			t.Fatalf("apply: %v", err)
		}
	}
	return len(changes)
}

func content(t *testing.T, path string) string {
	data, err := os.ReadFile(path) //nolint:gosec // a path of this test's own
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestCheckoutFirstRunThenAPlanFindsNoChange(t *testing.T) {
	c, _ := upstream(t, "v1.2.3")
	if n := run(t, c); n != 1 {
		t.Fatalf("first run planned %d changes", n)
	}
	if got := content(t, filepath.Join(c.Dir, "mise.toml")); got != "# v1.2.3\n" {
		t.Errorf("mise.toml = %q", got)
	}
	if n := run(t, c); n != 0 {
		t.Errorf("a plan after the run found %d changes", n)
	}
}

func TestCheckoutMovesToANewerTagAndDropsWhatGitDoesNotTrack(t *testing.T) {
	c, repo := upstream(t, "v1.2.3")
	run(t, c)
	release(t, repo, "v1.2.4")
	for name, body := range map[string]string{"stray": "an untracked file", "mise.local.toml": "pins another ws"} {
		if err := os.WriteFile(filepath.Join(c.Dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(c.Dir, "mise.toml"), []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if n := run(t, c); n != 1 {
		t.Errorf("a stray file and a change planned %d changes", n)
	}
	c.Tag = "v1.2.4"
	if n := run(t, c); n != 1 {
		t.Fatalf("a newer tag planned %d changes", n)
	}
	if got := content(t, filepath.Join(c.Dir, "mise.toml")); got != "# v1.2.4\n" {
		t.Errorf("mise.toml = %q", got)
	}
	for _, name := range []string{"stray", "mise.local.toml"} {
		if _, err := os.Stat(filepath.Join(c.Dir, name)); !os.IsNotExist(err) {
			t.Errorf("%s stays: %v", name, err)
		}
	}
	if n := run(t, c); n != 0 {
		t.Errorf("a plan after the run found %d changes", n)
	}
}

func TestCheckoutPlansTheCleanOfAnIgnoredFile(t *testing.T) {
	c, _ := upstream(t, "v1.2.3")
	run(t, c)
	if err := os.WriteFile(filepath.Join(c.Dir, "mise.local.toml"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if n := run(t, c); n != 1 {
		t.Errorf("an ignored file planned %d changes", n)
	}
}

func TestCheckoutTakesAnRC(t *testing.T) {
	c, _ := upstream(t, "v1.3.0-rc.2")
	if n := run(t, c); n != 1 {
		t.Fatalf("planned %d changes", n)
	}
	if got := content(t, filepath.Join(c.Dir, "mise.toml")); got != "# v1.3.0-rc.2\n" {
		t.Errorf("mise.toml = %q", got)
	}
}

func TestCheckoutInPlaceLeavesTheTreeAsItStands(t *testing.T) {
	c, _ := upstream(t, "v1.2.3")
	run(t, c)
	stray := filepath.Join(c.Dir, "stray")
	if err := os.WriteFile(stray, []byte("kept"), 0o600); err != nil {
		t.Fatal(err)
	}
	c.InPlace, c.Tag = true, ""
	if n := run(t, c); n != 0 {
		t.Errorf("--in-place planned %d changes", n)
	}
	if got := content(t, stray); got != "kept" {
		t.Errorf("stray = %q", got)
	}
}

func TestCheckoutRefusesADirectoryThatHoldsNoCheckout(t *testing.T) {
	c, _ := upstream(t, "v1.2.3")
	if err := os.Mkdir(c.Dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "holds no checkout") {
		t.Errorf("err = %v", err)
	}
}

func TestCheckoutRefusesAnythingButAReleaseTag(t *testing.T) {
	c, _ := upstream(t, "v1.2.3")
	for _, tag := range []string{"", "main", "v1.2", "v01.2.3", "v1.2.3+dirty", "v1.2.4-0.20260101-abc", "v1.2.3-rc.0"} {
		c.Tag = tag
		if _, err := c.Plan(t.Context()); err == nil {
			t.Errorf("%q was taken", tag)
		}
	}
}

func TestReleaseTag(t *testing.T) {
	for _, version := range []string{"v0.13.0", "v1.20.3", "v0.13.0-rc.5"} {
		if got, err := setup.ReleaseTag(version); err != nil || got != version {
			t.Errorf("%s: %q, %v", version, got, err)
		}
	}
	for _, version := range []string{"devel", "", "0.13.0", "v0.13.1-0.20261101120000-abcdef123456+dirty",
		"v0.13.0+dirty", "v0.13.0-rc.1+dirty", "v1.2", "v1.2.3.4"} {
		_, err := setup.ReleaseTag(version)
		if err == nil || !strings.Contains(err.Error(), "--in-place") {
			t.Errorf("%q: err = %v", version, err)
		}
	}
}

func TestCheckoutAsUserChangesTheTreeBySudoAndReadsItAsTheUser(t *testing.T) {
	c, _ := upstream(t, "v1.2.3")
	log := filepath.Join(t.TempDir(), "sudo.log")
	c.M = local.OSMachine{Sudo: []string{"sh", "-c", `printf '%s\n' "$*" >>"$0"; exec "$@"`, log}}
	c.AsUser = true
	if n := run(t, c); n != 1 {
		t.Fatalf("first run planned %d changes", n)
	}
	before := strings.Count(content(t, log), "\n")
	if n := run(t, c); n != 0 || strings.Count(content(t, log), "\n") != before {
		t.Errorf("a run after it planned %d changes, and sudo ran again", n)
	}
	for i, verb := range []string{"init", "fetch", "checkout", "clean"} {
		if line := strings.Split(content(t, log), "\n")[i]; !strings.HasPrefix(line, "git ") || !strings.Contains(line, " "+verb+" ") {
			t.Errorf("sudo's call %d = %q, want a git %s", i, line, verb)
		}
	}
	if before != 4 {
		t.Errorf("sudo ran %d times, want 4", before)
	}
}

func TestCheckoutAsUserNamesTheTreeSafeInEachRead(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/opt/ws"], m.Dirs["/opt/ws/.git"] = 0o755, 0o755
	m.Results["git --version"] = effecttest.Result{}
	c := setup.Checkout{M: m, Dir: "/opt/ws", Repo: "https://example.org/ws", Tag: "v1.2.3", AsUser: true}
	// The machine has no result for a read, which Plan takes for a tree to mend.
	if changes, err := c.Plan(t.Context()); err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	for _, call := range m.Calls {
		if call == "run git --version" || !strings.HasPrefix(call, "run git ") {
			continue
		}
		if want := "run git -c safe.directory=/opt/ws -C /opt/ws "; !strings.HasPrefix(call, want) {
			t.Errorf("call %q lacks the prefix %q", call, want)
		}
	}
}
