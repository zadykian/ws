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

// Hidden asks for a secret, with no echo. It stands beside Prompt, whose fakes in other modules go on
// compiling: effect is public.
type Hidden interface {
	// AskHidden returns the line as typed, blanks included.
	AskHidden(ctx context.Context, text string) (string, error)
}

// Out tells the maintainer what ws does.
type Out interface {
	Say(format string, a ...any)
	Warn(format string, a ...any)
	// Diff shows a unified diff of the lines of before and after.
	Diff(name string, before, after []byte)
	// Raw prints a line unchanged, with no prefix of the program, such as a line to copy.
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
	// Link is where the path points, or empty where the path is no link.
	Link string
}

// Lasting keeps a line past the screen that draws it. It stands beside Out, whose fakes in other
// modules go on compiling: effect is public.
type Lasting interface {
	SayLasting(format string, a ...any)
}

// SayLasting says a line that the maintainer acts on past a run, such as a key to copy. An out that
// doesn't draw a screen loses no line, so it takes a Say, and a plain run's output stays as before.
func SayLasting(out Out, format string, a ...any) {
	if l, ok := out.(Lasting); ok {
		l.SayLasting(format, a...)
		return
	}
	out.Say(format, a...)
}

// Machine reads and changes a machine.
type Machine interface {
	// Stat returns fs.ErrNotExist where the path is missing, and describes any link, one whose
	// target it cannot stat with no Mode.
	Stat(path string) (Info, error)
	// ReadDir returns the names in a directory, sorted, and fs.ErrNotExist where the directory is
	// missing.
	ReadDir(path string) ([]string, error)
	// ReadFile returns fs.ErrNotExist where the file is missing.
	ReadFile(path string) ([]byte, error)
	WriteFile(path string, data []byte, perm fs.FileMode) error
	// WriteRootFile replaces a file that root owns, through sudo and a new file renamed over it,
	// which is never readable beyond perm. It sets the owner to root.
	WriteRootFile(path string, data []byte, perm fs.FileMode) error
	// ReadRootFile reads a file that only root may read, and returns fs.ErrNotExist where the
	// file is missing.
	ReadRootFile(path string) ([]byte, error)
	// RootAttrs returns the owner and mode of a file, read through sudo, and fs.ErrNotExist where
	// the file is missing.
	RootAttrs(path string) (Attrs, error)
	// WriteAtomic replaces a file through a pending file, so that a reader sees the old file or the
	// new one whole, and sets the owner and mode of a. Another owner than ws's own needs root:
	// where ws runs as a user, sudo gives it outside the home and Confine, never through a link.
	WriteAtomic(path string, data []byte, a Attrs) error
	// Download fetches the URL into path through a pending file, which replaces path only where the
	// SHA-256 of the body, in hex, is sum. It sets the owner and mode of a.
	Download(ctx context.Context, url, path, sum string, a Attrs) error
	// Unpack makes dir at 0755 where missing, and unpacks the gzip tar archive into it, less the
	// first strip components of each name. It refuses an entry that leaves dir, and writes as
	// ws's own user: the server is root.
	Unpack(ctx context.Context, archive, dir string, strip int) error
	// Chown sets the owner of the file a link points to, through sudo as WriteAtomic's, where the
	// file or the owner is another's.
	Chown(path string, uid, gid int) error
	// Symlink makes path a link to target, and replaces a file or link there.
	Symlink(target, path string) error
	// Mkdir makes the directory with exactly perm, whatever the umask, through sudo as
	// WriteAtomic's, where ws may not write the parent.
	Mkdir(path string, perm fs.FileMode) error
	// Chmod sets the mode of the file a link points to, through sudo as WriteAtomic's, where the
	// file is another's.
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
	// Pass runs the command, through sudo where root is set. It passes its standard output and
	// error to ws's standard error, as a script's command with its output sent to stderr does.
	Pass(ctx context.Context, root bool, name string, args ...string) error
	// Exec replaces ws with the command, whose environment is ws's with env's entries set. It
	// returns only where it fails to start the command.
	Exec(name string, args, env []string) error
}

// Lines runs a command whose output ws reads as it comes. It stands beside Machine, as Foreground does.
type Lines interface {
	// PassLines is PassInput with each line of the output, without its newline, offered to take
	// first. A line that take takes stays off Stderr. take runs on a goroutine of the command's, a
	// call at a time, and every call ends before PassLines returns.
	PassLines(ctx context.Context, stdin []byte, take func(line string) bool, name string, args ...string) error
}

// Foreground runs a command that talks to the maintainer. It stands beside Machine, whose fakes in
// other modules go on compiling: effect is public.
type Foreground interface {
	// PassForeground is Pass on the terminal itself. A screen suspends first, and draws again after.
	PassForeground(ctx context.Context, root bool, name string, args ...string) error
}
