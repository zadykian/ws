// Command ws sets up the Mac that works on the server.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

// usageError is an error of the command line: a wrong command, flag or count of arguments.
type usageError struct{ err error }

func (e usageError) Error() string { return e.err.Error() }

func (e usageError) Unwrap() error { return e.err }

// signalExit is the error of a run that a signal cancelled. ws exits with 128 plus the signal's
// number, as a shell reports it.
type signalExit struct{ sig syscall.Signal }

func (e signalExit) Error() string { return e.sig.String() }

// group returns a command that only holds others. Alone it prints its help, and an unknown
// subcommand is a usage error.
func group(use, short string, cmds ...*cobra.Command) *cobra.Command {
	c := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE:  func(c *cobra.Command, _ []string) error { return c.Help() },
	}
	c.AddCommand(cmds...)
	return c
}

// newRoot returns the ws command with the groups of this system.
func newRoot(p process, groups ...*cobra.Command) *cobra.Command {
	root := group("ws", "set up the server and the Mac that works on it", newVersionCmd(p), newUpdateCmd(p))
	root.AddCommand(groups...)
	root.PersistentFlags().Bool(plainFlag, false, "print lines, with no full-screen display")
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return usageError{err} })
	walk(root, func(c *cobra.Command) {
		if args := c.Args; args != nil {
			c.Args = func(c *cobra.Command, a []string) error {
				if err := args(c, a); err != nil {
					return usageError{err}
				}
				return nil
			}
		}
	})
	return root
}

// walk calls f on c and on each command below it.
func walk(c *cobra.Command, f func(*cobra.Command)) {
	f(c)
	for _, sub := range c.Commands() {
		walk(sub, f)
	}
}

// execute runs root with args and returns the exit status. An error prints on stderr after the
// path of the command that returned it: a usage error with a line to the command's help, and
// status 2, any other error with status 1. A signal's exit prints nothing.
func execute(ctx context.Context, root *cobra.Command, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	root.SetArgs(append([]string{}, args...))
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)
	cmd, err := root.ExecuteContextC(ctx)
	if err == nil {
		return 0
	}
	var cut signalExit
	if errors.As(err, &cut) {
		return 128 + int(cut.sig)
	}
	path := cmd.CommandPath()
	for line := range strings.SplitSeq(err.Error(), "\n") {
		_, _ = fmt.Fprintf(stderr, "%s: %s\n", path, line)
	}
	var usage usageError
	if errors.As(err, &usage) {
		_, _ = fmt.Fprintf(stderr, "Run '%s --help' for usage.\n", path)
		return 2
	}
	return 1
}

func main() {
	p := newProcess()
	os.Exit(execute(context.Background(), newRoot(p, groups(p)...), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
