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
	"time"
)

// screenGrace bounds a command's end past its context under a screen. It stays above the 2 seconds
// that runuser gives its child, which a kill of runuser would cut short, and under the 5 seconds
// that a screen gives a run's cleanup.
const screenGrace = 3 * time.Second

// OSMachine is the Machine of the machine that ws runs on.
type OSMachine struct {
	// Sudo is the command that runs another as root. Nil means sudo; empty, for root itself, runs it as it is.
	Sudo []string
	// Owner is the uid:gid that root files get. Empty means 0:0; tests set their own.
	Owner string
	// Stderr takes what RunShown, PassInput and Pass pass on. Nil means os.Stderr.
	Stderr io.Writer
	// Stdin is what Pass gives a command to read. Nil means os.Stdin, where sudo can ask for a
	// password. A screen owns the terminal, and sets NoStdin. With any Stdin, git and ssh get
	// offTerminal, as they ask at /dev/tty, past their standard input, and a command's context ends
	// it by SIGTERM before a kill.
	Stdin io.Reader
	// Ticket, where set, runs before each command as root, which runs only where it passes. A screen
	// sets it beside a Sudo that can't ask, to take sudo's password in a form. It holds a run's
	// context, as WriteRootFile has none.
	Ticket func() error
	// Confine is a directory that Mkdir, Chmod, Chown, Symlink, WriteFile and WriteAtomic stay
	// inside for a path under it: a link there that leaves it fails the write. ws_user owns the
	// ws_home, and can swap a directory in it for a link as root applies, which would hand the
	// target to ws_user. Empty means no limit.
	Confine string
}

// ErrNoTicket ends a Ticket where sudo took the password and kept no ticket, as with
// timestamp_timeout=0. A sudo that can't ask then refuses each command, and --plain asks for each.
var ErrNoTicket = errors.New("sudo keeps no ticket here, so a screen can't run root's commands: run with --plain")

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

func (m OSMachine) ticket(root bool) error {
	if root && m.Ticket != nil {
		return m.Ticket()
	}
	return nil
}

// ReadFile reads the file.
func (OSMachine) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path) //nolint:gosec // the path is a step's own, not input
}

// WriteFile writes the file, and sets its mode where it already existed.
func (m OSMachine) WriteFile(path string, data []byte, perm fs.FileMode) error {
	if root, rel, err := m.confined(path); err != nil || root != nil {
		if err != nil {
			return err
		}
		defer func() { _ = root.Close() }()
		f, err := root.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
		if err != nil {
			return err
		}
		_, err = f.Write(data)
		if err = errors.Join(err, f.Chmod(perm), f.Close()); err != nil {
			return err
		}
		return nil
	}
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
	if err := m.ticket(true); err != nil {
		return nil, err
	}
	return output(m.command(ctx, append(append(m.sudo(), name), args...)))
}

// RunShown runs the command and returns its standard output. Its standard error goes to Stderr.
func (m OSMachine) RunShown(ctx context.Context, root bool, name string, args ...string) ([]byte, error) {
	if err := m.ticket(root); err != nil {
		return nil, err
	}
	cmd := m.command(ctx, m.argv(root, name, args))
	cmd.Stderr = m.stderr()
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
	if err := m.ticket(root); err != nil {
		return err
	}
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

func (m OSMachine) command(ctx context.Context, argv []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...) //nolint:gosec // the steps name the command
	if m.Stdin != nil {
		cmd.Env = withEnv(os.Environ(), offTerminal)
		// A raw terminal sends the command's group no Ctrl-C, so ctx's end is all that reaches it. A
		// kill would end runuser and leave its child, whose hold on a screen's pipe holds up Wait.
		cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
		cmd.WaitDelay = screenGrace
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
