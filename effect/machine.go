package effect

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// OSMachine is the Machine of the machine that ws runs on.
type OSMachine struct {
	// Sudo is the command that runs another as root. Nil means sudo; empty, for root itself, runs it as it is.
	Sudo []string
	// Owner is the uid:gid that root files get. Empty means 0:0; tests set their own.
	Owner string
	// Stderr takes what RunShown, PassInput and Pass pass on. Nil means os.Stderr.
	Stderr io.Writer
	// Confine is a directory that Mkdir, Chmod, Chown, Symlink, WriteFile and WriteAtomic stay
	// inside for a path under it: a link there that leaves it fails the write. ws_user owns the
	// ws_home, and can swap a directory in it for a link as root applies, which would hand the
	// target to ws_user. Empty means no limit.
	Confine string
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

// RunShown runs the command and returns its standard output. Its standard error goes to Stderr.
func (m OSMachine) RunShown(ctx context.Context, root bool, name string, args ...string) ([]byte, error) {
	cmd := m.command(ctx, m.argv(root, name, args))
	cmd.Stderr = m.stderr()
	return output(cmd)
}

// PassInput runs the command with stdin as its standard input, and its standard output and error
// sent to Stderr.
func (m OSMachine) PassInput(ctx context.Context, stdin []byte, name string, args ...string) error {
	cmd := m.command(ctx, append([]string{name}, args...))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = bytes.NewReader(stdin), m.stderr(), m.stderr()
	if err := cmd.Run(); err != nil {
		return commandError(cmd, err)
	}
	return nil
}

// Pass runs the command with its standard output and error sent to Stderr, and standard input kept,
// so that sudo can ask for a password.
func (m OSMachine) Pass(ctx context.Context, root bool, name string, args ...string) error {
	cmd := m.command(ctx, m.argv(root, name, args))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, m.stderr(), m.stderr()
	if err := cmd.Run(); err != nil {
		return commandError(cmd, err)
	}
	return nil
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

func (OSMachine) command(ctx context.Context, argv []string) *exec.Cmd {
	return exec.CommandContext(ctx, argv[0], argv[1:]...) //nolint:gosec // the steps name the command
}

func output(cmd *exec.Cmd) ([]byte, error) {
	out, err := cmd.Output()
	if err != nil {
		return out, commandError(cmd, err)
	}
	return out, nil
}

func commandError(cmd *exec.Cmd, err error) error {
	var ee *exec.ExitError
	if errors.As(err, &ee) && len(ee.Stderr) > 0 {
		return fmt.Errorf("%v: %w: %s", cmd.Args, err, bytes.TrimSpace(ee.Stderr))
	}
	return fmt.Errorf("%v: %w", cmd.Args, err)
}

// Stat describes the path. A link that points nowhere has a Link and no Mode.
func (OSMachine) Stat(path string) (Info, error) {
	var info Info
	if l, err := os.Readlink(path); err == nil {
		info.Link = l
	}
	fi, err := os.Stat(path)
	if err != nil {
		if info.Link != "" && errors.Is(err, fs.ErrNotExist) {
			return info, nil
		}
		return Info{}, err
	}
	info.Mode = fi.Mode()
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

// Mkdir makes the directory with exactly perm, whatever the umask.
func (m OSMachine) Mkdir(path string, perm fs.FileMode) error {
	if root, rel, err := m.confined(path); err != nil || root != nil {
		if err != nil {
			return err
		}
		defer func() { _ = root.Close() }()
		if err := root.Mkdir(rel, perm); err != nil {
			return err
		}
		return m.Chmod(path, perm)
	}
	if err := os.Mkdir(path, perm); err != nil {
		return err
	}
	return m.Chmod(path, perm)
}

// Chmod sets the mode of the file a link points to.
func (m OSMachine) Chmod(path string, perm fs.FileMode) error {
	root, rel, err := m.confined(path)
	if err != nil {
		return err
	}
	if root == nil {
		return os.Chmod(path, perm)
	}
	defer func() { _ = root.Close() }()
	return root.Chmod(rel, perm)
}
