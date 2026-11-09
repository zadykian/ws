package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/internal/tui"
)

// plainFlag names the persistent flag that keeps a command to lines.
const plainFlag = "plain"

// jsonFlag names the flag that prints a command's data.
const jsonFlag = "json"

// drawsScreen tells whether a command draws a full-screen display: stdin and stderr are
// terminals, TERM names one that can draw, and neither --plain nor --json is given. Anywhere else
// ws prints lines, which CI, pipes and the unit on ws read.
func drawsScreen(c *cobra.Command, p process) bool {
	plain, _ := c.Flags().GetBool(plainFlag)
	// A command without --json has no such flag, and the lookup then reads false.
	asJSON, _ := c.Flags().GetBool(jsonFlag)
	return screenWanted(plain, asJSON, p.tty, p.errTTY, p.getenv("TERM"))
}

// drawsReport tells whether a command draws the report that it would print on stdout. That takes a
// terminal there too: a pipe or a file waits for the text, which a screen never prints.
func drawsReport(c *cobra.Command, p process) bool {
	return p.outTTY && drawsScreen(c, p)
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

// screenConfig is what a screen draws with: ws's lines go to stderr, where it draws. Its works run
// on p's machine.
func screenConfig(c *cobra.Command, p process) tui.Config {
	return tui.Config{
		In: c.InOrStdin(), Out: c.ErrOrStderr(), Theme: tui.NewTheme(p.getenv), Machine: p.machine,
		Cut: func(s syscall.Signal) error { return signalExit{s} },
	}
}

// drawRun runs work beside a run screen of groups, on stderr, where the lines go, and returns once
// the terminal is back. work reads Bridge.Stop between its steps. A run cut short ends with the
// status of a signal, for execute to report.
func drawRun(c *cobra.Command, p process, title string, groups []tui.Group, work tui.Work) error {
	cfg, done := runConfig(c, p, screenConfig(c, p))
	defer done()
	return tui.Run(c.Context(), cfg, tui.NewRunScreen(title, cfg.Theme, groups...), work)
}

// runConfig returns cfg as a run draws it: it asks before it stops, and keeps a log, which done
// closes once the run has ended.
func runConfig(c *cobra.Command, p process, cfg tui.Config) (tui.Config, func()) {
	log, path := openLog(p, c.CommandPath(), time.Now())
	cfg.Halts, cfg.LogPath = true, path
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
