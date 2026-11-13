package lazygit

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// wrapped says how run builds the wrapper's surroundings. Its lazygit is the stub's body, which
// replaces the wrapper as lazygit does. Its terminal drops the test for a terminal, as a test's
// stdin and stdout are pipes. Its tmux, where set, names the log of a stub for tmux in PATH, and
// sets TMUX.
type wrapped struct {
	dir, lazygit, tmux string
	terminal           bool
}

// run runs the wrapper in w.dir, with a stub for lazygit that prints its arguments and exits 3,
// and a watcher that looks every 0.1 seconds. It returns the output and the exit status.
func run(t *testing.T, w wrapped) (out string, code int) {
	t.Helper()
	bin := t.TempDir()
	write := func(path, body string) {
		if err := os.WriteFile(path, []byte("#!/bin/sh\n"+body), 0o755); err != nil { //nolint:gosec // a test's stub runs
			t.Fatal(err)
		}
	}
	stub := filepath.Join(bin, "lazygit")
	write(stub, w.lazygit+"\necho \"lazygit $*\"\nexit 3\n")
	// The ceiling keeps git from finding a repository that holds the test's directories.
	env := []string{"PATH=" + bin + ":/usr/bin:/bin", "HOME=" + bin, "GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CEILING_DIRECTORIES=" + os.TempDir()}
	if w.tmux != "" {
		write(filepath.Join(bin, "tmux"), "echo \"$*\" >>"+w.tmux+"\n")
		env = append(env, "TMUX=/tmp/s,1,0", "TMUX_PANE=%1")
	}
	script := strings.ReplaceAll(string(wrapper), "/usr/bin/lazygit", stub)
	script = strings.ReplaceAll(script, "sleep 1", "sleep 0.1")
	if w.terminal {
		script = strings.ReplaceAll(script, "[ ! -t 0 ] || [ ! -t 1 ]", "false")
	}
	path := filepath.Join(bin, "wrapper")
	write(path, script)
	cmd := exec.CommandContext(t.Context(), "/bin/sh", path, "a", "b c") //nolint:gosec // the test's own script
	cmd.Dir, cmd.Env = w.dir, env
	// The watcher holds the output open until it sees lazygit gone.
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

// linux skips a test of the watcher, which reads lazygit's directory from /proc.
func linux(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "linux" {
		t.Skip("the watcher reads /proc")
	}
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

func TestTheWrapperFollowsACheckout(t *testing.T) {
	linux(t)
	dir := repository(t, filepath.Join(t.TempDir(), "ws"))
	gitIn(t, dir, "branch", "feat/x")
	out, _ := run(t, wrapped{dir: dir, terminal: true, lazygit: "git switch -q feat/x\nsleep 0.5"})
	if want := push + osc("ws: main") + osc("ws: feat/x") + "lazygit a b c\n" + pop; out != want {
		t.Errorf("out %q, want %q", out, want)
	}
}

func TestTheWrapperFollowsAnotherRepository(t *testing.T) {
	linux(t)
	root := t.TempDir()
	dir := repository(t, filepath.Join(root, "ws"))
	other := repository(t, filepath.Join(root, "cld"))
	out, _ := run(t, wrapped{dir: dir, terminal: true, lazygit: "cd " + other + "\nsleep 0.5"})
	if want := push + osc("ws: main") + osc("cld: main") + "lazygit a b c\n" + pop; out != want {
		t.Errorf("out %q, want %q", out, want)
	}
}

func TestTheWrapperNamesALinkedWorktreeByItsMainTree(t *testing.T) {
	root := t.TempDir()
	dir := repository(t, filepath.Join(root, "ws"))
	tree := filepath.Join(root, "ws", ".claude", "worktrees", "fix")
	gitIn(t, dir, "worktree", "add", "-q", "-b", "fix/1", tree)
	out, _ := run(t, wrapped{dir: tree, terminal: true})
	if !strings.Contains(out, osc("ws: fix/1")) {
		t.Errorf("out %q", out)
	}
}

func TestTheWrapperNamesADetachedHeadByItsHash(t *testing.T) {
	dir := repository(t, filepath.Join(t.TempDir(), "ws"))
	gitIn(t, dir, "switch", "-q", "--detach")
	out, _ := run(t, wrapped{dir: dir, terminal: true})
	if want := osc("ws: " + gitIn(t, dir, "rev-parse", "--short", "HEAD")); !strings.Contains(out, want) {
		t.Errorf("out %q, want %q in it", out, want)
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

func TestTheWrapperSetsThePanesTitleInTmux(t *testing.T) {
	linux(t)
	dir := repository(t, filepath.Join(t.TempDir(), "ws"))
	gitIn(t, dir, "branch", "next")
	log := filepath.Join(t.TempDir(), "tmux")
	out, _ := run(t, wrapped{dir: dir, terminal: true, tmux: log, lazygit: "git switch -q next\nsleep 0.5"})
	if out != push+"lazygit a b c\n"+pop {
		t.Errorf("out %q", out)
	}
	logged, err := os.ReadFile(log) //nolint:gosec // the test's own log
	if err != nil {
		t.Fatal(err)
	}
	if want := "select-pane -t %1 -T ws: main\nselect-pane -t %1 -T ws: next\n"; string(logged) != want {
		t.Errorf("tmux got %q, want %q", logged, want)
	}
}
