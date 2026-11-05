package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

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
	out := c.ErrOrStderr()
	log, path := openLog(p, c.CommandPath(), time.Now())
	if log != nil {
		defer func() { _ = log.Close() }()
	}
	cfg := tui.Config{
		In: c.InOrStdin(), Out: out, Theme: theme, Halts: true, LogPath: path,
		Cut: func(s syscall.Signal) error { return signalExit{s} },
	}
	if log != nil { // A nil *os.File in the interface would not be nil.
		cfg.Log = log
	}
	return tui.Run(c.Context(), cfg, tui.NewRunScreen(title, theme, groups...), work)
}

// openLog creates the log of a run of command, under $XDG_STATE_HOME/ws or ~/.local/state/ws, for
// the owner alone. A log that can't be made is no reason to stop a run, so it returns nil.
func openLog(p process, command string, now time.Time) (*os.File, string) {
	dir := p.getenv("XDG_STATE_HOME")
	if dir == "" {
		if home := p.getenv("HOME"); home != "" {
			dir = filepath.Join(home, ".local", "state")
		}
	}
	if !filepath.IsAbs(dir) {
		return nil, ""
	}
	dir = filepath.Join(dir, "ws")
	if os.MkdirAll(dir, 0o700) != nil {
		return nil, ""
	}
	name := strings.ReplaceAll(command, " ", "-") + "-" + now.UTC().Format("20060102T150405Z") + ".log"
	path := filepath.Join(dir, name)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec // under the state directory, named by ws
	if err != nil {
		return nil, ""
	}
	return f, path
}

// screenMachine returns m as a run's steps take it under a screen. A fake has no terminal to keep
// its commands off.
func screenMachine(ctx context.Context, b *tui.Bridge, m effect.Machine) effect.Machine {
	if machine, ok := m.(effect.OSMachine); ok {
		return b.Machine(ctx, machine)
	}
	return m
}
