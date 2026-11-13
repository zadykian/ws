package engine

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/zadykian/ws/engine/tui"
)

// DrawsScreen tells whether a command draws a full-screen display: stdin and stderr are
// terminals, TERM names one that can draw, and neither --plain nor --json is given. Anywhere else
// ws prints lines, which CI, pipes and the unit on ws read.
func DrawsScreen(c Call) bool {
	return screenWanted(c.Plain, c.JSON, c.TTY, c.ErrTTY, c.Getenv("TERM"))
}

// DrawsReport tells whether a command draws the report that it would print on stdout. That takes a
// terminal there too: a pipe or a file waits for the text, which a screen never prints.
func DrawsReport(c Call) bool {
	return c.OutTTY && DrawsScreen(c)
}

// screenWanted is the choice itself, on its five inputs.
func screenWanted(plain, asJSON, stdinTTY, stderrTTY bool, term string) bool {
	return !plain && !asJSON && stdinTTY && stderrTTY && term != "" && term != "dumb"
}

// runTitle is a run screen's title: the command, with --check where it makes no change.
func runTitle(c Call, check bool) string {
	if check {
		return c.Path + " --check"
	}
	return c.Path
}

// ScreenConfig is what a screen draws with: ws's lines go to stderr, where it draws, and a run's
// stdout waits for the screen's end. Its works run on the call's machine.
func ScreenConfig(c Call) tui.Config {
	return tui.Config{
		In: c.In, Out: c.Err, Text: c.Out, Theme: tui.NewTheme(c.Getenv),
		Machine: c.Machine, Cut: func(s syscall.Signal) error { return SignalExit{Sig: s} },
	}
}

// drawRun runs work beside a run screen of groups, on stderr, where the lines go, and returns once
// the terminal is back. work gives its steps Bridge.Options. A run cut short ends with the
// status of a signal, for execute to report.
func drawRun(ctx context.Context, c Call, title string, groups []tui.Group, work tui.Work) error {
	cfg, done := runConfig(c, ScreenConfig(c))
	defer done()
	return tui.Run(ctx, cfg, tui.NewRunScreen(title, cfg.Theme, groups...), work)
}

// runConfig returns cfg as a run draws it: it asks before it stops, and keeps a log, which done
// closes once the run has ended.
func runConfig(c Call, cfg tui.Config) (tui.Config, func()) {
	log, path := openLog(c, time.Now())
	cfg.LogPath = path
	if log == nil { // A nil *os.File in the interface would not be nil.
		return cfg, func() {}
	}
	cfg.Log = log
	return cfg, func() { _ = log.Close() }
}

// logsKept is how many logs stay in the state directory, the newest.
const logsKept = 20

// openLog creates the log of a run of the call's command, under $XDG_STATE_HOME/ws or ~/.local/state/ws, for
// the owner alone. A log that can't be made is no reason to stop a run, so it returns nil.
func openLog(c Call, now time.Time) (*os.File, string) {
	dir := c.Getenv("XDG_STATE_HOME")
	if dir == "" {
		if home := c.Getenv("HOME"); home != "" {
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
	name := strings.ReplaceAll(c.Path, " ", "-") + "-" + now.UTC().Format("20060102T150405Z") + ".log"
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
