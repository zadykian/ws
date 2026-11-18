package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
)

// Group returns a command that only holds others. Alone it prints its help, and an unknown
// subcommand is a usage error.
func Group(use, short string, cmds ...*cobra.Command) *cobra.Command {
	c := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE:  func(c *cobra.Command, _ []string) error { return c.Help() },
	}
	c.AddCommand(cmds...)
	return c
}

// Root returns the ws command with the groups of this system.
func Root(p Env, groups ...*cobra.Command) *cobra.Command {
	root := Group("ws", "set up the server and the Mac that works on it", Version(p))
	root.AddCommand(groups...)
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return engine.UsageError{Err: err} })
	Walk(root, func(c *cobra.Command) {
		if args := c.Args; args != nil {
			c.Args = func(c *cobra.Command, a []string) error {
				if err := args(c, a); err != nil {
					return engine.UsageError{Err: err}
				}
				return nil
			}
		}
	})
	return root
}

// Walk calls f on c and on each command below it.
func Walk(c *cobra.Command, f func(*cobra.Command)) {
	f(c)
	for _, sub := range c.Commands() {
		Walk(sub, f)
	}
}

// Execute runs root with args and returns the exit status. An error prints on stderr after the
// path of the command that returned it. A usage error adds a line to the command's help and takes
// status 2, an ExitStatus its own code, and any other error status 1. A signal's exit prints
// nothing, nor does an ExitStatus without an error of its own.
func Execute(ctx context.Context, root *cobra.Command, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	root.SetArgs(append([]string{}, args...))
	root.SetIn(stdin)
	root.SetOut(stdout)
	root.SetErr(stderr)
	cmd, err := root.ExecuteContextC(ctx)
	if err == nil {
		return 0
	}
	var cut engine.SignalExit
	if errors.As(err, &cut) {
		return 128 + int(cut.Sig)
	}
	path := cmd.CommandPath()
	var status engine.ExitStatus
	if errors.As(err, &status) {
		if status.Err != nil {
			printLines(stderr, path, err)
		}
		return status.Code
	}
	printLines(stderr, path, err)
	var usage engine.UsageError
	if errors.As(err, &usage) {
		_, _ = fmt.Fprintf(stderr, "Run '%s --help' for usage.\n", path)
		return 2
	}
	return 1
}

// printLines prints each line of err on w, after the path of the command that returned it.
func printLines(w io.Writer, path string, err error) {
	for line := range strings.SplitSeq(err.Error(), "\n") {
		_, _ = fmt.Fprintf(w, "%s: %s\n", path, line)
	}
}
