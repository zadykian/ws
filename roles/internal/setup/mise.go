// Package setup holds the first steps of setting a machine up, which the server and the Mac
// share. They are mise at the release ws pins, and the files in /etc/mise that name the machine's
// side and trust ws's tree. Each step is a resource, so that a plan reports it before any write.
package setup

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/fetch"
)

// MisePath is where ws puts mise's binary.
const MisePath = "/usr/local/bin/mise"

// MiseRelease is a release of mise: its version and the SHA-256 of its binary for each target,
// in hex.
type MiseRelease struct {
	Version string
	Sums    map[string]string
}

// PinnedMise is the release ws installs. Where mise moves, take the sums from the release's
// SHASUMS256.txt. The targets are the release's asset names.
var PinnedMise = MiseRelease{
	Version: "2026.11.9",
	Sums: map[string]string{
		"linux-x64":   "f13f3e5ec88807e0161669b3aa07a91d84478cdf1e95b5b92b63f85a25d42395",
		"linux-arm64": "b8dd05af6447cee933928664982a8cb2fa646e6951cfc8539db23ce2554a7b04",
	},
}

// URL returns where the release's binary for the target is downloaded.
func (r MiseRelease) URL(target string) string {
	return fmt.Sprintf("https://github.com/jdx/mise/releases/download/v%s/mise-v%s-%s", r.Version, r.Version, target)
}

// MiseBinary is mise at MisePath, at the version of its Release. It plans nothing where
// mise --version names that version. Else it plans a download, checked against the SHA-256 that the
// release holds for Target, as a first run and a drift of the pin both need.
type MiseBinary struct {
	M       effect.Machine
	Release MiseRelease
	// Target names the release's binary for this machine, as linux-x64.
	Target string
}

// Plan reads mise's version, and plans the download where it differs.
func (b MiseBinary) Plan(ctx context.Context) ([]resource.Change, error) {
	sum, ok := b.Release.Sums[b.Target]
	if !ok {
		return nil, fmt.Errorf("no SHA-256 of mise %s is pinned for %s", b.Release.Version, b.Target)
	}
	if out, err := b.M.Run(ctx, MisePath, "--version"); err == nil &&
		strings.HasPrefix(string(out), b.Release.Version+" ") {
		return nil, nil
	}
	return fetch.Download{M: b.M, URL: b.Release.URL(b.Target), Path: MisePath, SHA256: sum, Mode: 0o755}.Plan(ctx)
}
