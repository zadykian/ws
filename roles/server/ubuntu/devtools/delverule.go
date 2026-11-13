package devtools

import (
	"bytes"
	"slices"
	"strings"
)

// delveModule is Delve's module, as go version -m names it.
const delveModule = "github.com/go-delve/delve"

// delveCurrent reports whether go version -m's output and error describe Delve at the version,
// built with the Go version. A failed read is no Delve. The first line ends in the Go that built
// the binary, and a mod line carries the module and its version.
func delveCurrent(out []byte, err error, goVersion, version string) bool {
	if err != nil {
		return false
	}
	lines := strings.Split(string(bytes.TrimSpace(out)), "\n")
	if !strings.HasSuffix(lines[0], " go"+goVersion) {
		return false
	}
	return slices.ContainsFunc(lines[1:], func(line string) bool {
		f := strings.Fields(line)
		return len(f) >= 3 && f[0] == "mod" && f[1] == delveModule && f[2] == "v"+version
	})
}
