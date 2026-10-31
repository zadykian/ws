package effect

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
)

// OSMachine is the Machine of the Mac that ws runs on.
type OSMachine struct {
	// Sudo is the command that runs another as root. Nil means sudo.
	Sudo []string
	// Owner is the uid:gid that root files get. Empty means 0:0; tests set their own.
	Owner string
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
func (OSMachine) WriteFile(path string, data []byte, perm fs.FileMode) error {
	if err := os.WriteFile(path, data, perm); err != nil {
		return err
	}
	return os.Chmod(path, perm)
}

// Run runs the command and returns its standard output.
func (m OSMachine) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	return output(m.command(ctx, append([]string{name}, args...)))
}

// RunRoot runs the command through sudo.
func (m OSMachine) RunRoot(ctx context.Context, name string, args ...string) ([]byte, error) {
	return output(m.command(ctx, append(append(m.sudo(), name), args...)))
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
