package lazygit

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The title's escapes: the stack's push and pop, and OSC 2.
const (
	push = "\033[22;2t"
	pop  = "\033[23;2t"
)

func osc(title string) string { return "\033]2;" + title + "\007" }

// gitIn runs git in dir with no config of the machine's or the user's, and returns its output. Of
// the test's environment it takes PATH alone, as a rebase's exec sets GIT_DIR to the outer one.
func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "git", append([]string{"-C", dir}, args...)...) //nolint:gosec // the test's own git
	cmd.Env = []string{"PATH=" + os.Getenv("PATH"), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t"}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// repository makes a repository at dir with one commit on main.
func repository(t *testing.T, dir string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "init", "-q", "-b", "main")
	gitIn(t, dir, "commit", "-q", "--allow-empty", "-m", "first")
	return dir
}

// wrapped says how run builds the wrapper's surroundings: terminal drops the test for a terminal,
// as a test's stdin and stdout are pipes.
type wrapped struct {
	dir      string
	terminal bool
}

// run runs the wrapper in w.dir, with a stub for lazygit that prints its arguments and exits 3,
// and returns the output and the exit status.
func run(t *testing.T, w wrapped) (out string, code int) {
	t.Helper()
	bin := t.TempDir()
	write := func(path, body string) {
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil { //nolint:gosec // a test's stub runs
			t.Fatal(err)
		}
	}
	stub := filepath.Join(bin, "lazygit")
	write(stub, "echo \"lazygit $*\"\nexit 3\n")
	// The ceiling keeps git from finding a repository that holds the test's directories.
	env := []string{"PATH=" + bin + ":/usr/bin:/bin", "HOME=" + bin, "GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CEILING_DIRECTORIES=" + os.TempDir()}
	script := strings.ReplaceAll(string(wrapper), "/usr/bin/lazygit", stub)
	if w.terminal {
		script = strings.ReplaceAll(script, "[ ! -t 0 ] || [ ! -t 1 ]", "false")
	}
	path := filepath.Join(bin, "wrapper")
	write(path, script)
	cmd := exec.CommandContext(t.Context(), "/bin/sh", path, "a", "b c") //nolint:gosec // the test's own script
	cmd.Dir, cmd.Env = w.dir, env
	cmd.WaitDelay = 5 * time.Second
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err := cmd.Run()
	var ee *exec.ExitError
	if !errors.As(err, &ee) {
		t.Fatalf("run: %v", err)
	}
	return stdout.String(), ee.ExitCode()
}

func TestTheWrapperPassesAPipeThrough(t *testing.T) {
	out, code := run(t, wrapped{dir: t.TempDir()})
	if out != "lazygit a b c\n" || code != 3 {
		t.Errorf("out %q, exit %d", out, code)
	}
}

func TestTheWrapperNamesTheRepositoryAndBranch(t *testing.T) {
	dir := repository(t, filepath.Join(t.TempDir(), "ws"))
	out, code := run(t, wrapped{dir: dir, terminal: true})
	if want := push + osc("ws: main") + "lazygit a b c\n" + pop; out != want {
		t.Errorf("out %q, want %q", out, want)
	}
	if code != 3 {
		t.Errorf("exit %d, want lazygit's", code)
	}
}

func TestTheWrapperSetsNoTitleOutsideARepository(t *testing.T) {
	out, code := run(t, wrapped{dir: t.TempDir(), terminal: true})
	if out != push+"lazygit a b c\n"+pop || code != 3 {
		t.Errorf("out %q, exit %d", out, code)
	}
}

func TestTheWrapperDropsControlCharacters(t *testing.T) {
	dir := repository(t, filepath.Join(t.TempDir(), "w\033]0;x\007s"))
	out, _ := run(t, wrapped{dir: dir, terminal: true})
	if want := push + osc("w]0;xs: main") + "lazygit a b c\n" + pop; out != want {
		t.Errorf("out %q, want %q", out, want)
	}
}
