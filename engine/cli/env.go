// Package cli defines ws's commands that no side owns, over the process they run in.
package cli

import (
	"os"
	"path/filepath"
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
	// Pwd is the directory a relative WS_VALUES is taken from.
	Pwd string
	// Version is this ws's version, as ws version prints it.
	Version string
	// Self returns the file this ws runs from, which the kinds name as {{ self }}.
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
		Pwd:       pwd,
		Version:   versionOf(info, ok),
		Self:      runningFile,
	}
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
