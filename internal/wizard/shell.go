package wizard

import (
	"context"
	"strings"
)

// Shell runs a command as root on ws, with in as its standard input, for a Session: SSHShell from
// the Mac, or LocalShell on ws itself. A command's words reach it as they are.
type Shell interface {
	// Run returns the command's standard output.
	Run(ctx context.Context, in []byte, cmd ...string) ([]byte, error)
	// Pass passes the command's output to Stderr as it comes, for one that goes on, such as install.
	Pass(ctx context.Context, in []byte, cmd ...string) error
	// Lines is Pass with each line of the output offered to take first.
	Lines(ctx context.Context, in []byte, take func(line string) bool, cmd ...string) error
}

// SSHShell runs a command on ws over ssh, which reaches ws at the address that the Host key stage
// asked, where it asked one, and reaches root by Sudo.
type SSHShell struct {
	SSH SSH
	// Sudo is how a command reaches root. Nil runs it as the login, as the tests do.
	Sudo *Sudo
}

// args returns ssh's arguments and stdin for cmd as root, with in as cmd's stdin. ssh hands ws's
// shell one line, so a word that its shell would read otherwise goes quoted.
func (s SSHShell) args(cmd []string, in []byte) ([]string, []byte, error) {
	cmd, in, err := s.Sudo.rooted(cmd, in)
	args := []string{"-o", "BatchMode=yes", "ws"}
	for _, word := range cmd {
		args = append(args, shellWord(word))
	}
	return args, in, err
}

// Run runs cmd on ws, and returns its standard output.
func (s SSHShell) Run(ctx context.Context, in []byte, cmd ...string) ([]byte, error) {
	args, in, err := s.args(cmd, in)
	if err != nil {
		return nil, err
	}
	return s.SSH.RunInput(ctx, in, args...)
}

// Pass is Run with the output passed to Stderr as it comes.
func (s SSHShell) Pass(ctx context.Context, in []byte, cmd ...string) error {
	args, in, err := s.args(cmd, in)
	if err != nil {
		return err
	}
	return s.SSH.PassInput(ctx, in, args...)
}

// Lines is Pass with each line offered to take first.
func (s SSHShell) Lines(ctx context.Context, in []byte, take func(line string) bool, cmd ...string) error {
	args, in, err := s.args(cmd, in)
	if err != nil {
		return err
	}
	return s.SSH.PassLines(ctx, in, take, args...)
}

// shellWord returns word as sh reads it back: bare where it holds only letters, digits and
// -_./=:,+@. Else it goes in single quotes, with each quote of its own closed, escaped and opened
// again.
func shellWord(word string) string {
	if word != "" && strings.Trim(word, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_./=:,+@") == "" {
		return word
	}
	return "'" + strings.ReplaceAll(word, "'", `'\''`) + "'"
}
