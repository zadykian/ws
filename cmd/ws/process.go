package main

import (
	"os"
	"runtime"
	"runtime/debug"

	"github.com/zadykian/ws/effect"
)

// process is what the commands take from the process they run in. main builds it from os, and the
// tests build it from fakes. The streams and the context come from cobra.
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
	// version is this ws's version, as ws version prints it.
	version string
	// argv is ws's arguments, without the program, which ws mac wizard --update runs the new ws with.
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
		version:  versionOf(info, ok),
		argv:     os.Args[1:],
		releases: releasesURL(os.Getenv),
		self:     runningFile,
	}
}
