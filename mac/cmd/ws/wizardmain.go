package main

import (
	"cmp"
	"context"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/zadykian/ws/mac/internal/effect"
)

// confPath returns mac.conf's path from WS_CONF, a path from / as ws mac apply gets it, else from the
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

// wizardMain runs ws mac wizard in the process: with /opt/homebrew/bin first in PATH, as a shell
// started before Homebrew's install lacks it, and WS_CONF set for the scripts that stages run.
func wizardMain(args []string, stderr io.Writer) int {
	pwd, _ := os.Getwd()
	info, ok := debug.ReadBuildInfo()
	e := wizardEnv{goos: runtime.GOOS, uid: os.Geteuid(), pid: os.Getpid(), m: effect.OSMachine{}, version: versionOf(info, ok), releases: releasesURL(os.Getenv),
		getenv: os.Getenv, lookup: os.LookupEnv, pwd: pwd, stdin: os.Stdin, tty: isTerminal(os.Stdin), stderr: stderr, stdout: os.Stdout}
	_ = os.Setenv("PATH", "/opt/homebrew/bin:"+os.Getenv("PATH"))
	_ = os.Setenv("WS_CONF", confPath(os.Getenv, pwd))
	return wizardCommand(context.Background(), args, e)
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
