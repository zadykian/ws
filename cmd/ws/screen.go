package main

import (
	"context"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/tui"
)

// plainFlag names the persistent flag that keeps a command to lines.
const plainFlag = "plain"

// drawsScreen tells whether a command draws a full-screen display: stdin and stderr are
// terminals, TERM names one that can draw, and --plain is not given. Anywhere else ws prints
// lines, which CI, pipes and the unit on ws read.
func drawsScreen(c *cobra.Command, p process) bool {
	plain, _ := c.Flags().GetBool(plainFlag)
	return screenWanted(plain, p.tty, p.errTTY, p.getenv("TERM"))
}

// screenWanted is the choice itself, on its four inputs.
func screenWanted(plain, stdinTTY, stderrTTY bool, term string) bool {
	return !plain && stdinTTY && stderrTTY && term != "" && term != "dumb"
}

// runTitle is a run screen's title: the command, with --check where it makes no change.
func runTitle(c *cobra.Command, check bool) string {
	if check {
		return c.CommandPath() + " --check"
	}
	return c.CommandPath()
}

// drawRun runs work beside a run screen of groups, on stderr, where the lines go, and returns once
// the terminal is back. work reads Bridge.Stop between its steps. A run cut short ends with the
// status of a signal, for execute to report.
func drawRun(c *cobra.Command, p process, title string, groups []tui.Group, work tui.Work) error {
	theme := tui.NewTheme(p.getenv)
	return tui.Run(c.Context(), tui.Config{
		In: c.InOrStdin(), Out: c.ErrOrStderr(), Theme: theme, Halts: true,
		Cut: func(s syscall.Signal) error { return signalExit{s} },
	}, tui.NewRunScreen(title, theme, groups...), work)
}

// screenMachine returns m as a run's steps take it under a screen. A fake has no terminal to keep
// its commands off.
func screenMachine(ctx context.Context, b *tui.Bridge, m effect.Machine) effect.Machine {
	if machine, ok := m.(effect.OSMachine); ok {
		return b.Machine(ctx, machine)
	}
	return m
}
