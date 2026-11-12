package cmd

import (
	"cmp"
	"os"
	"strings"
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
