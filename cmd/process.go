package cmd

import (
	"cmp"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/zadykian/ws/engine/core/effect"
)

// process is what the commands take from the process they run in. Execute builds it from os, and
// the tests build it from fakes. The streams and the context come from cobra.
type process struct {
	machine effect.Machine
	getenv  func(string) string
	// lookup tells a variable set to nothing from one that is not set.
	lookup func(string) (string, bool)
	setenv func(key, value string) error
	euid   int
	pid    int
	goos   string
	// goarch names a release's binary for this machine, with goos.
	goarch string
	// pwd is the directory a relative WS_CONF is taken from.
	pwd string
	// tty tells whether stdin is a terminal, which the questions need.
	tty bool
	// openTTY opens the terminal, /dev/tty, to ask a password past a stdin that is not one. Nil has none.
	openTTY func() (effect.Hidden, func(), error)
	// errTTY tells whether stderr is a terminal, where a screen draws.
	errTTY bool
	// outTTY tells whether stdout is a terminal, where a report prints.
	outTTY bool
	// version is this ws's version, as ws version prints it.
	version string
	// argv is ws's arguments, without the program, which ws client macos wizard --update runs the new
	// ws with.
	argv []string
	// releases is the address of ws's releases, WS_RELEASES_URL or GitHub's.
	releases string
	// self returns the file this ws runs from, which ws update replaces.
	self func() (string, error)
}

// newProcess returns the process of this ws.
func newProcess() process {
	info, ok := debug.ReadBuildInfo()
	pwd, _ := os.Getwd()
	return process{
		machine:  effect.OSMachine{},
		getenv:   os.Getenv,
		lookup:   os.LookupEnv,
		setenv:   os.Setenv,
		euid:     os.Geteuid(),
		pid:      os.Getpid(),
		goos:     runtime.GOOS,
		goarch:   runtime.GOARCH,
		pwd:      pwd,
		tty:      isTerminal(os.Stdin),
		openTTY:  openTTY,
		errTTY:   isTerminal(os.Stderr),
		outTTY:   isTerminal(os.Stdout),
		version:  versionOf(info, ok),
		argv:     os.Args[1:],
		releases: releasesURL(os.Getenv),
		self:     runningFile,
	}
}

// lockDir is the directory that the wizards' lock goes in: TMPDIR, or /tmp where TMPDIR is empty.
func (p process) lockDir() string { return cmp.Or(p.getenv("TMPDIR"), "/tmp") }

// openTTY opens /dev/tty, which asks with no echo and prints its question there.
func openTTY() (effect.Hidden, func(), error) {
	f, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return nil, nil, err
	}
	return effect.NewTerminal(f, f), func() { _ = f.Close() }, nil
}
