// Package effect holds the effects ws runs through: what it asks, says and does on a machine. Steps
// take these interfaces, so that their tests use the fakes of effecttest.
package effect

import (
	"context"
	"io/fs"
)

// Question is what ws asks, with its default and its check of the answer.
type Question struct {
	Text    string
	Default string
	// Valid is pure. A nil Valid takes any answer.
	Valid func(string) error
}

// Prompt asks the maintainer.
type Prompt interface {
	// Ask asks again while q.Valid fails.
	Ask(ctx context.Context, q Question) (string, error)
}

// Out tells the maintainer what ws does.
type Out interface {
	Say(format string, a ...any)
	Warn(format string, a ...any)
	// Diff shows a unified diff of the lines of before and after.
	Diff(name string, before, after []byte)
	// Raw prints a line as it is, with no prefix of the program, such as a line to copy.
	Raw(format string, a ...any)
	// ByHand keeps a line, and prints it once the run ends.
	ByHand(format string, a ...any)
}

// Attrs are the owner and the permission bits of a file.
type Attrs struct {
	UID, GID int
	Mode     fs.FileMode
}

// Info is what Stat finds of a path.
type Info struct {
	// Mode is the mode of the file a link points to, with its type bits.
	Mode fs.FileMode
	// Link is where the path points, or empty where it is no link.
	Link string
}

// Machine reads and changes a machine.
type Machine interface {
	// Stat returns fs.ErrNotExist where the path is missing, and describes a dangling link.
	Stat(path string) (Info, error)
	// ReadDir returns the names in a directory, sorted, and fs.ErrNotExist where it is missing.
	ReadDir(path string) ([]string, error)
	// ReadFile returns fs.ErrNotExist where the file is missing.
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	// WriteRootFile replaces a file that root owns, through sudo and a new file renamed over it,
	// which is never readable beyond perm. It sets the owner to root.
	WriteRootFile(path string, data []byte, perm fs.FileMode) error
	// ReadRootFile reads a file that only root may read, and returns fs.ErrNotExist where it is
	// missing.
	ReadRootFile(path string) ([]byte, error)
	// RootAttrs returns the owner and mode of a file, read through sudo, and fs.ErrNotExist where
	// it is missing.
	RootAttrs(path string) (Attrs, error)
	// Mkdir makes the directory with exactly perm, whatever the umask.
	Mkdir(path string, perm fs.FileMode) error
	// Chmod sets the mode of the file a link points to.
	Chmod(path string, perm fs.FileMode) error
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
	// RunRoot runs the command through sudo.
	RunRoot(ctx context.Context, name string, args ...string) ([]byte, error)
	// RunInput is Run with the bytes as standard input.
	RunInput(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, error)
	// RunShown is Run, or RunRoot where root is set, with the command's standard error passed to
	// ws's own, as a shell script lets it through.
	RunShown(ctx context.Context, root bool, name string, args ...string) ([]byte, error)
	// PassInput is Run with the bytes as standard input, and with standard output and error sent to
	// Stderr as they come, for a command that goes on for minutes.
	PassInput(ctx context.Context, stdin []byte, name string, args ...string) error
	// Pass runs the command, through sudo where root is set, and passes its standard output and
	// error to ws's standard error, as a script's command with its output sent to stderr does.
	Pass(ctx context.Context, root bool, name string, args ...string) error
	// Exec replaces ws with the command, whose environment is ws's with env's entries set. It
	// returns only where it fails to start the command.
	Exec(name string, args, env []string) error
}
