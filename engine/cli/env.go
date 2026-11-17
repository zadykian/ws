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
	// Pwd is the directory a relative WS_VALUES is taken from.
	Pwd string
	// Version is this ws's version, as ws version prints it.
	Version string
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
		Version:  versionOf(info, ok),
		Releases: releasesURL(os.Getenv),
		Self:     runningFile,
	}
}

// releasesURL returns WS_RELEASES_URL, or GitHub's page of ws's releases, as install.sh does.
func releasesURL(getenv func(string) string) string {
	return cmp.Or(getenv("WS_RELEASES_URL"), "https://github.com/zadykian/ws/releases")
}
