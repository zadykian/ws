package wizard

import (
	"regexp"
	"strings"
)

// Homebrew's release: its Homebrew.pkg, that file's SHA-256, and its signer, as pkgutil names it.
const (
	brewVersion = "7.0.10"
	brewURL     = "https://github.com/Homebrew/brew/releases/download/" + brewVersion + "/Homebrew.pkg"
	brewSHA256  = "ae13b8b0ac910730d3e982a9e7518288d17fd0ad272e42f412027e8ab4555575"
	brewSigner  = "Developer ID Installer: Patrick Linnane (927JGANW46)"
)

var (
	pkgStatus = regexp.MustCompile(`^\s*Status: .* issued by Apple for distribution$`)
	pkgFirst  = regexp.MustCompile(`^\s*1\. (.*)$`)
)

// pkgSigner returns the signer in the output of pkgutil --check-signature: the first of the chain
// that it prints, after a status of Apple's Developer ID. It returns "" where there is none.
func pkgSigner(out string) string {
	seen := false
	for _, line := range strings.Split(out, "\n") {
		if pkgStatus.MatchString(line) {
			seen = true
		} else if m := pkgFirst.FindStringSubmatch(line); seen && m != nil {
			return m[1]
		}
	}
	return ""
}
