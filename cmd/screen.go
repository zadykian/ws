package cmd

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/tui"
)

// jsonFlag names the flag that prints a command's data.
const jsonFlag = "json"

// drawsScreen tells whether a command draws a full-screen display: stdin and stderr are
// terminals, TERM names one that can draw, and neither --plain nor --json is given. Anywhere else
// ws prints lines, which CI, pipes and the unit on ws read.
func drawsScreen(c *cobra.Command, p cli.Env) bool {
	plain, _ := c.Flags().GetBool(cli.PlainFlag)
	// A command without --json has no such flag, and the lookup then reads false.
	asJSON, _ := c.Flags().GetBool(jsonFlag)
	return screenWanted(plain, asJSON, p.TTY, p.ErrTTY, p.Getenv("TERM"))
}

// drawsReport tells whether a command draws the report that it would print on stdout. That takes a
// terminal there too: a pipe or a file waits for the text, which a screen never prints.
func drawsReport(c *cobra.Command, p cli.Env) bool {
	return p.OutTTY && drawsScreen(c, p)
}

// screenWanted is the choice itself, on its five inputs.
func screenWanted(plain, asJSON, stdinTTY, stderrTTY bool, term string) bool {
	return !plain && !asJSON && stdinTTY && stderrTTY && term != "" && term != "dumb"
}

// runTitle is a run screen's title: the command, with --check where it makes no change.
func runTitle(c *cobra.Command, check bool) string {
	if check {
		return c.CommandPath() + " --check"
	}
	return c.CommandPath()
}

// screenConfig is what a screen draws with: ws's lines go to stderr, where it draws, and a run's
// stdout waits for the screen's end. Its works run on p's machine.
func screenConfig(c *cobra.Command, p cli.Env) tui.Config {
	return tui.Config{
		In: c.InOrStdin(), Out: c.ErrOrStderr(), Text: c.OutOrStdout(), Theme: tui.NewTheme(p.Getenv),
		Machine: p.Machine, Cut: func(s syscall.Signal) error { return engine.SignalExit{Sig: s} },
	}
}

// drawRun runs work beside a run screen of groups, on stderr, where the lines go, and returns once
// the terminal is back. work gives its steps Bridge.Options. A run cut short ends with the
// status of a signal, for execute to report.
func drawRun(c *cobra.Command, p cli.Env, title string, groups []tui.Group, work tui.Work) error {
	cfg, done := runConfig(c, p, screenConfig(c, p))
	defer done()
	return tui.Run(c.Context(), cfg, tui.NewRunScreen(title, cfg.Theme, groups...), work)
}

// runConfig returns cfg as a run draws it: it asks before it stops, and keeps a log, which done
// closes once the run has ended.
func runConfig(c *cobra.Command, p cli.Env, cfg tui.Config) (tui.Config, func()) {
	log, path := openLog(p, c.CommandPath(), time.Now())
	cfg.LogPath = path
	if log == nil { // A nil *os.File in the interface would not be nil.
		return cfg, func() {}
	}
	cfg.Log = log
	return cfg, func() { _ = log.Close() }
}

// logsKept is how many logs stay in the state directory, the newest.
const logsKept = 20

// openLog creates the log of a run of command, under $XDG_STATE_HOME/ws or ~/.local/state/ws, for
// the owner alone. A log that can't be made is no reason to stop a run, so it returns nil.
func openLog(p cli.Env, command string, now time.Time) (*os.File, string) {
	dir := p.Getenv("XDG_STATE_HOME")
	if dir == "" {
		if home := p.Getenv("HOME"); home != "" {
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
	pruneLogs(dir)
	name := strings.ReplaceAll(command, " ", "-") + "-" + now.UTC().Format("20060102T150405Z") + ".log"
	path := filepath.Join(dir, name)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600) //nolint:gosec // under the state directory, named by ws
	if err != nil {
		return nil, ""
	}
	return f, path
}

// pruneLogs removes the oldest logs of dir, so that room is left for one more.
func pruneLogs(dir string) {
	logs, _ := filepath.Glob(filepath.Join(dir, "*.log"))
	age := map[string]time.Time{}
	for _, l := range logs {
		if info, err := os.Stat(l); err == nil {
			age[l] = info.ModTime()
		}
	}
	slices.SortFunc(logs, func(a, b string) int { return age[a].Compare(age[b]) })
	for len(logs) >= logsKept {
		_ = os.Remove(logs[0])
		logs = logs[1:]
	}
}
