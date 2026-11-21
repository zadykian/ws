// Package cli defines ws's commands that no side owns, over the process they run in.
package cli

import (
	"cmp"
	"os"
	"path/filepath"
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
	// LookupEnv tells a variable set but empty from one the run never set, as a run of the
	// secrets task sets each secret input, empty where it has no value.
	LookupEnv func(string) (string, bool)
	Euid      int
	// Pid names the copy that ws update checks beside the file it replaces.
	Pid  int
	GOOS string
	// GOARCH names a release's binary for this machine, with GOOS.
	GOARCH string
	// Pwd is the directory a relative WS_VALUES is taken from.
	Pwd string
	// Version is this ws's version, as ws version prints it.
	Version string
	// Releases is the address of ws's releases, WS_RELEASES_URL or GitHub's.
	Releases string
	// Self returns the file this ws runs from, which the kinds name as {{ self }}, and which ws update
	// runs setup from where this ws is the release.
	Self func() (string, error)
}

// NewEnv returns the Env of this ws.
func NewEnv() Env {
	info, ok := debug.ReadBuildInfo()
	pwd, _ := os.Getwd()
	return Env{
		Machine:   local.OSMachine{},
		Getenv:    os.Getenv,
		LookupEnv: os.LookupEnv,
		Euid:      os.Geteuid(),
		Pid:       os.Getpid(),
		GOOS:      runtime.GOOS,
		GOARCH:    runtime.GOARCH,
		Pwd:       pwd,
		Version:   versionOf(info, ok),
		Releases:  releasesURL(os.Getenv),
		Self:      runningFile,
	}
}

// releasesURL returns WS_RELEASES_URL, or GitHub's page of ws's releases, as install.sh does.
func releasesURL(getenv func(string) string) string {
	return cmp.Or(getenv("WS_RELEASES_URL"), "https://github.com/zadykian/ws/releases")
}

// TaskEnv returns LookupEnv in the run of the mise task named task, and else a lookup that finds
// nothing. mise names the task by MISE_TASK_NAME in the env of each process the task starts, its
// nested bootstrap's ws among them. So a side's inputs count in its task's run alone, never where a
// shell holds a stale value of one. An empty task names none.
func (p Env) TaskEnv(task string) func(string) (string, bool) {
	if task == "" || p.Getenv("MISE_TASK_NAME") != task {
		return func(string) (string, bool) { return "", false }
	}
	return p.LookupEnv
}

// runningFile returns the file this ws runs from, the file a symbolic link leads to.
func runningFile() (string, error) {
	file, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(file)
}
