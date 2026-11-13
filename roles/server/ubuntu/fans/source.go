package fans

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource/fetch"
)

// sourceDir is where DKMS finds the driver's source. The tree is staged beside it.
const sourceDir = "/usr/src/nct6687d-" + nct6687Short

// sourceTree is the driver's source at its commit, from GitHub's tarball, unpacked in /usr/src for
// DKMS. The tarball is the same for every architecture, so its pin is under "all".
func sourceTree(m effect.Machine) fetch.Tree {
	return fetch.Tree{M: m, Path: sourceDir, Strip: 1, Source: fetch.PinnedFile{
		M: m, Version: nct6687Commit, SHA256: map[string]string{"all": nct6687SHA256},
		URL: func(commit, _ string) string {
			return "https://github.com/Fred78290/nct6687d/archive/" + commit + ".tar.gz"
		},
	}}
}
