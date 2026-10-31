package main

import (
	"context"
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/zadykian/ws/mac/internal/effect"
)

// confPath returns mac.conf's path from WS_CONF, a path from / as setup.sh gets it, else from the
// home.
func confPath(getenv func(string) string, pwd string) string {
	switch file := getenv("WS_CONF"); {
	case file == "":
		return getenv("HOME") + "/" + confFile
	case strings.HasPrefix(file, "/"):
		return file
	default:
		return pwd + "/" + file
	}
}

// wizardMain runs ws wizard in the process: with /opt/homebrew/bin first in PATH, as a shell
// started before Homebrew's install lacks it, and WS_CONF set for the scripts that stages run.
func wizardMain(args []string, stderr io.Writer) int {
	pwd, _ := os.Getwd()
	e := wizardEnv{goos: runtime.GOOS, uid: os.Geteuid(), pid: os.Getpid(), m: effect.OSMachine{},
		getenv: os.Getenv, stdin: os.Stdin, stderr: stderr}
	_ = os.Setenv("PATH", "/opt/homebrew/bin:"+os.Getenv("PATH"))
	_ = os.Setenv("WS_CONF", confPath(os.Getenv, pwd))
	return wizardCommand(context.Background(), args, e)
}
