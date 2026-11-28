package local

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
)

// killGrace bounds a command's end past its context. A command with a child, such as sudo, has
// this long to end it after SIGTERM, which a kill of the parent would cut short.
const killGrace = 3 * time.Second

// OSMachine is the Machine of the machine that ws runs on.
type OSMachine struct {
	// Sudo is the command that runs another as root. Nil means sudo; empty, for root itself, runs it bare.
	Sudo []string
	// Owner is the uid:gid that root files get. Empty means 0:0; tests set their own.
	Owner string
	// Stderr takes what RunShown, PassInput and Pass pass on. Nil means os.Stderr.
	Stderr io.Writer
	// Stdin is what Pass gives a command to read. Nil means os.Stdin, where sudo can ask for a
	// password. With any Stdin, git and ssh get
	// offTerminal, as they ask at /dev/tty, past their standard input, and a command's context ends
	// it by SIGTERM before a kill.
	Stdin io.Reader
	// UserReads reads root's files and their owner and mode as ws's own user, never through sudo,
	// so that a read asks no password. A file the user may not read fails the read. The Mac's
	// units set it, as they keep root's paths readable by all; writes still go through sudo.
	UserReads bool
}

// NoStdin is a Stdin that ends at once, as http.NoBody is a body.
var NoStdin io.Reader = noStdin{}

type noStdin struct{}

func (noStdin) Read([]byte) (int, error) { return 0, io.EOF }

func (m OSMachine) stdin() io.Reader {
	if m.Stdin == nil {
		return os.Stdin
	}
	return m.Stdin
}

func (m OSMachine) stderr() io.Writer {
	if m.Stderr == nil {
		return os.Stderr
	}
	return m.Stderr
}

func (m OSMachine) sudo() []string {
	if m.Sudo == nil {
		return []string{"sudo", "--"}
	}
	return m.Sudo
}

// ReadFile reads the file.
func (OSMachine) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:gosec // the path is a step's own, not input
}

// WriteFile writes the file, and sets its mode where it already existed.
func (m OSMachine) WriteFile(path string, data []byte, perm fs.FileMode) error {
	if err := os.WriteFile(path, data, perm); err != nil {
		return err
	}
	return os.Chmod(path, perm)
}

// Run runs the command and returns its standard output.
func (m OSMachine) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return output(m.command(ctx, append([]string{name}, args...)))
}

// RunInput runs the command with stdin as its standard input and returns its standard output.
func (m OSMachine) RunInput(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, error) {
	cmd := m.command(ctx, append([]string{name}, args...))
	cmd.Stdin = bytes.NewReader(stdin)
	return output(cmd)
}

// RunRoot runs the command through sudo.
func (m OSMachine) RunRoot(ctx context.Context, name string, args ...string) ([]byte, error) {
	return output(m.command(ctx, append(append(m.sudo(), name), args...)))
}

// RunRootInput runs the command through sudo with stdin as its standard input.
func (m OSMachine) RunRootInput(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, error) {
	cmd := m.command(ctx, append(append(m.sudo(), name), args...))
	cmd.Stdin = bytes.NewReader(stdin)
	return output(cmd)
}

// RunShown runs the command and returns its standard output. Its standard error goes to Stderr.
func (m OSMachine) RunShown(ctx context.Context, root bool, name string, args ...string) ([]byte, error) {
	cmd := m.command(ctx, m.argv(root, name, args))
	cmd.Stderr = m.stderr()
	return output(cmd)
}

// RunTerminal runs the command with Stdin as its standard input and its standard error sent to
// Stderr, and returns its standard output.
func (m OSMachine) RunTerminal(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := m.command(ctx, append([]string{name}, args...))
	cmd.Stdin, cmd.Stderr = m.stdin(), m.stderr()
	return output(cmd)
}

// PassInput runs the command with stdin as its standard input, and its standard output and error
// sent to Stderr.
func (m OSMachine) PassInput(ctx context.Context, stdin []byte, name string, args ...string) error {
	cmd := m.command(ctx, append([]string{name}, args...))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = bytes.NewReader(stdin), m.stderr(), m.stderr()
	return finished(cmd, cmd.Run())
}

// Pass runs the command with its standard output and error sent to Stderr, and Stdin as its
// standard input.
func (m OSMachine) Pass(ctx context.Context, root bool, name string, args ...string) error {
	cmd := m.command(ctx, m.argv(root, name, args))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = m.stdin(), m.stderr(), m.stderr()
	return finished(cmd, cmd.Run())
}

// Exec replaces ws with the command, found on PATH, with ws's environment and env's entries set.
func (OSMachine) Exec(name string, args, env []string) error {
	path, err := exec.LookPath(name)
	if err != nil {
		return err
	}
	return syscall.Exec(path, append([]string{name}, args...), withEnv(os.Environ(), env)) //nolint:gosec // the steps name the command
}

// withEnv returns base with each KEY=VALUE entry of set added, replacing the entry of its name.
func withEnv(base, set []string) []string {
	name := func(e string) string { k, _, _ := strings.Cut(e, "="); return k }
	replaced := map[string]bool{}
	for _, e := range set {
		replaced[name(e)] = true
	}
	out := make([]string, 0, len(base)+len(set))
	for _, e := range base {
		if !replaced[name(e)] {
			out = append(out, e)
		}
	}
	return append(out, set...)
}

func (m OSMachine) argv(root bool, name string, args []string) []string {
	if root {
		return append(append(m.sudo(), name), args...)
	}
	return append([]string{name}, args...)
}

// offTerminal makes git and ssh fail where they would ask, as a clone's ssh does of an unknown host.
// An askpass that fails gives ssh an empty answer. A GIT_SSH_COMMAND with BatchMode would do the same,
// but would replace a core.sshCommand.
var offTerminal = []string{"GIT_TERMINAL_PROMPT=0", "SSH_ASKPASS=false", "SSH_ASKPASS_REQUIRE=force"}

// repoVars name a repository to git, as git rev-parse --local-env-vars lists them. A git before
// 2.40 lists GIT_INTERNAL_SUPER_PREFIX too. A hook or a rebase's exec exports GIT_DIR, which sends
// a command's git to the caller's repository. There, an init makes it again, and a checkout moves
// its HEAD and replaces its index. ws's commands act on the paths that they name, so none gets
// these.
var repoVars = []string{"GIT_ALTERNATE_OBJECT_DIRECTORIES", "GIT_CONFIG", "GIT_CONFIG_PARAMETERS",
	"GIT_CONFIG_COUNT", "GIT_OBJECT_DIRECTORY", "GIT_DIR", "GIT_WORK_TREE", "GIT_IMPLICIT_WORK_TREE",
	"GIT_GRAFT_FILE", "GIT_INDEX_FILE", "GIT_NO_REPLACE_OBJECTS", "GIT_REPLACE_REF_BASE", "GIT_PREFIX",
	"GIT_INTERNAL_SUPER_PREFIX", "GIT_SHALLOW_FILE", "GIT_COMMON_DIR"}

// environ is ws's environment less repoVars.
func environ() []string {
	return slices.DeleteFunc(os.Environ(), func(e string) bool {
		name, _, _ := strings.Cut(e, "=")
		return slices.Contains(repoVars, name)
	})
}

func (m OSMachine) command(ctx context.Context, argv []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...) //nolint:gosec // the steps name the command
	cmd.Env = environ()
	if m.Stdin != nil {
		cmd.Env = withEnv(cmd.Env, offTerminal)
		// A raw terminal sends the command's group no Ctrl-C, so ctx's end is all that reaches it. A
		// kill would end sudo and leave its child, whose hold on a pipe holds up Wait.
		cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
		cmd.WaitDelay = killGrace
	}
	return cmd
}

func output(cmd *exec.Cmd) ([]byte, error) {
	out, err := cmd.Output()
	return out, finished(cmd, err)
}

// finished is the error of cmd, past what Run or Output returned. ErrWaitDelay is none: the command
// passed, and left a process that holds its output, which a plain run's file never waits on.
func finished(cmd *exec.Cmd, err error) error {
	if err == nil || errors.Is(err, exec.ErrWaitDelay) {
		return nil
	}
	return commandError(cmd, err)
}

func commandError(cmd *exec.Cmd, err error) error {
	var ee *exec.ExitError
	if errors.As(err, &ee) && len(ee.Stderr) > 0 {
		return fmt.Errorf("%v: %w: %s", cmd.Args, err, bytes.TrimSpace(ee.Stderr))
	}
	return fmt.Errorf("%v: %w", cmd.Args, err)
}

// Stat describes the path. A link whose target it cannot stat has a Link and no Mode, whatever the
// error. A guard that looks for links must see one that points into a directory it cannot search.
func (OSMachine) Stat(path string) (effect.Info, error) {
	var info effect.Info
	if l, err := os.Readlink(path); err == nil {
		info.Link = l
	}
	fi, err := os.Stat(path)
	if err != nil {
		if info.Link != "" {
			return info, nil
		}
		return effect.Info{}, err
	}
	info.Mode = fi.Mode()
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		info.UID, info.GID = int(st.Uid), int(st.Gid)
	}
	return info, nil
}

// ReadDir returns the names in the directory, sorted.
func (OSMachine) ReadDir(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.Name()
	}
	return names, nil
}

// Mkdir makes the directory with exactly perm, whatever the umask, through sudo where ws runs as
// a user that may not write the parent.
func (m OSMachine) Mkdir(path string, perm fs.FileMode) error {
	if m.asRoot(path) && unwritable(filepath.Dir(path)) {
		return m.sudoMkdir(path, perm)
	}
	if err := os.Mkdir(path, perm); err != nil {
		return err
	}
	return m.Chmod(path, perm)
}
