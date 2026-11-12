package ikeward

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/host"
)

// version is ikeward's, from its release's binary. sums are the SHA-256 of that binary for each
// architecture, as the release's ikeward.sha256 gives it.
const version = "0.3.0"

var sums = map[string]string{
	"amd64": "b78ba9456f246ddaa5f08f34706cf96b975ad3717a69f142cbcbd893294b2d23",
	"arm64": "711b9a1c72b59ce7bc453d94c790855588b529803b8789abc72ed3b405e83e3b",
}

// binary is where install puts ikeward, and where its services, the routing's and strongSwan run
// it: install refuses to run as any other program.
const binary = "/usr/local/sbin/ikeward"

const releases = "https://github.com/zadykian/ikeward/releases"

// binaryURL is the release's binary for the architecture, as dpkg names it.
func binaryURL(version, arch string) string {
	return releases + "/download/v" + version + "/ikeward-linux-" + arch
}

// binaryFile makes the release's binary without install.sh, where install.sh would put it. Its
// download is checked against the SHA-256 before it writes the file, and made again where the
// file holds another, a former version's.
func binaryFile(m effect.Machine) host.PinnedFile {
	return host.PinnedFile{M: m, Version: version, URL: binaryURL, SHA256: sums, Path: binary, Mode: 0o755}
}
