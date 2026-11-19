// Package launchd runs launchctl for a LaunchDaemon of macOS, a job of launchd's system domain. It
// reads a daemon's state as the user, which launchctl allows, and loads, unloads, restarts and
// signals it through sudo.
package launchd

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
)

// launchctl's exit statuses that a caller tells apart.
const (
	// notFound is print's, where launchd has no job of the label.
	notFound = 113
	// noProcess is bootout's, where the job is not loaded.
	noProcess = 3
	// ioError is bootstrap's, where the job is loaded still, or the plist is refused.
	ioError = 5
)

// bootstrapTries bounds a bootstrap's tries, a second apart.
const bootstrapTries = 10

// bootoutTries bounds the wait for a daemon to go after its bootout, a read a second. launchd gives
// a job 20 s to exit before it kills it.
const bootoutTries = 30

// Loaded is what launchctl print shows of a daemon.
type Loaded struct {
	// Found is false where launchd has no daemon of the label.
	Found bool
	// Arguments are the program and its arguments as launchd loaded them, which an edit of the
	// plist leaves until a bootout and a bootstrap. They are nil where print shows none.
	Arguments []string
	// PID is the running process's, 0 where none runs.
	PID int
}

// Launchctl calls launchctl on M for the daemons of the system domain.
type Launchctl struct {
	M effect.Machine
	// Sleep waits between the tries of a bootstrap, and the reads after a bootout. Nil means
	// time.Sleep.
	Sleep func(time.Duration)
}

// target is a daemon's name to launchctl.
func target(label string) string { return "system/" + label }

// exitStatus is the exit status of a command's error, or -1 for one that did not exit.
func exitStatus(err error) int {
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode()
	}
	return -1
}

// Print reads the daemon's state as the user, who needs no root for it.
func (l Launchctl) Print(ctx context.Context, label string) (Loaded, error) {
	out, err := l.M.Run(ctx, "launchctl", "print", target(label))
	if exitStatus(err) == notFound {
		return Loaded{}, nil
	}
	if err != nil {
		return Loaded{}, err
	}
	return parsePrint(string(out)), nil
}

// parsePrint reads print's keys one tab deep: pid, and the block of arguments, one a line two tabs
// deep up to its closing brace. print's form is no stable interface, so it reads no other key.
func parsePrint(out string) Loaded {
	l := Loaded{Found: true}
	in := false
	for line := range strings.Lines(out) {
		line = strings.TrimRight(line, "\r\n")
		switch {
		case in && line == "\t}":
			in = false
		case in:
			l.Arguments = append(l.Arguments, strings.TrimPrefix(line, "\t\t"))
		case line == "\targuments = {" && l.Arguments == nil:
			in = true
		case strings.HasPrefix(line, "\tpid = "):
			l.PID, _ = strconv.Atoi(strings.TrimPrefix(line, "\tpid = "))
		}
	}
	return l
}

// Bootstrap loads the daemon from its plist, which starts it where the plist says RunAtLoad. A
// daemon booted out a moment before may be on its way out still, so it tries for 10 s while
// launchctl answers exit 5.
func (l Launchctl) Bootstrap(ctx context.Context, plist string) error {
	for tries := 1; ; tries++ {
		_, err := l.M.RunRoot(ctx, "launchctl", "bootstrap", "system", plist)
		switch {
		case err == nil:
			return nil
		case exitStatus(err) != ioError || tries >= bootstrapTries:
			return fmt.Errorf("launchctl bootstrap system %s: %w; launchd refuses a plist that is not "+
				"root's at mode 644, and a daemon that System Settings › General › Login Items & "+
				"Extensions does not allow in the background", plist, err)
		}
		l.pause()
	}
}

// pause waits a second, by Sleep where set.
func (l Launchctl) pause() {
	if l.Sleep == nil {
		time.Sleep(time.Second)
		return
	}
	l.Sleep(time.Second)
}

// Bootout unloads the daemon and stops it. A daemon that is not loaded is no error. launchctl
// returns while the daemon may still be on its way out, and may still write its files. So Bootout
// waits up to 30 s, until print no longer finds it.
func (l Launchctl) Bootout(ctx context.Context, label string) error {
	_, err := l.M.RunRoot(ctx, "launchctl", "bootout", target(label))
	if err != nil && exitStatus(err) != noProcess {
		return err
	}
	for tries := 1; ; tries++ {
		loaded, err := l.Print(ctx, label)
		switch {
		case err != nil:
			return err
		case !loaded.Found:
			return nil
		case tries >= bootoutTries:
			return fmt.Errorf("launchd still has %s %d s after its bootout", target(label), bootoutTries)
		}
		l.pause()
	}
}

// Kickstart stops the daemon's process and starts it again, so that it runs a new program.
func (l Launchctl) Kickstart(ctx context.Context, label string) error {
	_, err := l.M.RunRoot(ctx, "launchctl", "kickstart", "-k", target(label))
	return err
}

// Signal sends the daemon's process sig, such as SIGHUP. launchctl refuses it where the daemon
// runs no process.
func (l Launchctl) Signal(ctx context.Context, label, sig string) error {
	_, err := l.M.RunRoot(ctx, "launchctl", "kill", sig, target(label))
	return err
}
