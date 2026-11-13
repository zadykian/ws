// Package cli defines ws's commands that no side owns, over the process they run in.
package cli

import (
	"cmp"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/local"
)

// Env is what the commands take from the process they run in. NewEnv builds it from os, and the
// tests build it from fakes. The streams and the context come from cobra.
type Env struct {
	Machine effect.Machine
	Getenv  func(string) string
	// Lookup tells a variable set to nothing from one that is not set.
	Lookup func(string) (string, bool)
	Setenv func(key, value string) error
	Euid   int
	Pid    int
	GOOS   string
	// GOARCH names a release's binary for this machine, with GOOS.
	GOARCH string
	// Pwd is the directory a relative WS_CONF is taken from.
	Pwd string
	// TTY tells whether stdin is a terminal, which the questions need.
	TTY bool
	// OpenTTY opens the terminal, /dev/tty, to ask a password past a stdin that is not one. Nil has none.
	OpenTTY func() (effect.Hidden, func(), error)
	// ErrTTY tells whether stderr is a terminal, where a screen draws.
	ErrTTY bool
	// OutTTY tells whether stdout is a terminal, where a report prints.
	OutTTY bool
	// Version is this ws's version, as ws version prints it.
	Version string
	// Argv is ws's arguments, without the program, which ws client macos wizard --update runs the new
	// ws with.
	Argv []string
	// Releases is the address of ws's releases, WS_RELEASES_URL or GitHub's.
	Releases string
	// Self returns the file this ws runs from, which ws update replaces.
	Self func() (string, error)
}

// NewEnv returns the Env of this ws.
func NewEnv() Env {
	info, ok := debug.ReadBuildInfo()
	pwd, _ := os.Getwd()
	return Env{
		Machine:  local.OSMachine{},
		Getenv:   os.Getenv,
		Lookup:   os.LookupEnv,
		Setenv:   os.Setenv,
		Euid:     os.Geteuid(),
		Pid:      os.Getpid(),
		GOOS:     runtime.GOOS,
		GOARCH:   runtime.GOARCH,
		Pwd:      pwd,
		TTY:      isTerminal(os.Stdin),
		OpenTTY:  openTTY,
		ErrTTY:   isTerminal(os.Stderr),
		OutTTY:   isTerminal(os.Stdout),
		Version:  versionOf(info, ok),
		Argv:     os.Args[1:],
		Releases: releasesURL(os.Getenv),
		Self:     runningFile,
	}
}

// LockDir is the directory that the wizards' lock goes in: TMPDIR, or /tmp where TMPDIR is empty.
func (e Env) LockDir() string { return cmp.Or(e.Getenv("TMPDIR"), "/tmp") }

// openTTY opens /dev/tty, which asks with no echo and prints its question there.
func openTTY() (effect.Hidden, func(), error) {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}
	return effect.NewTerminal(f, f), func() { _ = f.Close() }, nil
}

// isTerminal tells a terminal from a pipe, a file and /dev/null, as sh's [ -t 0 ] does.
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return false
	}
	null, err := os.Stat(os.DevNull)
	return err != nil || !os.SameFile(info, null)
}

// releasesURL returns WS_RELEASES_URL, or GitHub's page of ws's releases, as install.sh does.
func releasesURL(getenv func(string) string) string {
	return cmp.Or(getenv("WS_RELEASES_URL"), "https://github.com/zadykian/ws/releases")
}
