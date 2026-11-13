// Package golang installs Go, and gopls and Delve built with it.
package golang

import (
	"context"
	"path"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource/fetch"
	"github.com/zadykian/ws/roles/server/ubuntu/devtools/internal/web"
)

// goDir is where Go is installed. Its tree is staged beside it, so that a run cut short never
// leaves a half tree there.
const goDir = "/usr/local/go"

// Tree is Go's tarball of the version, from go.dev, unpacked in /usr/local/go. dpkg's name of the
// architecture is Go's. go.dev's list gives the SHA-256, so a plan that finds the tree current
// fetches nothing.
func Tree(m effect.Machine, version string) fetch.Tree {
	return fetch.Tree{M: m, Path: goDir, Strip: 1, Source: fetch.PinnedFile{
		M: m, Version: version,
		URL: func(version, arch string) string {
			return "https://go.dev/dl/go" + version + ".linux-" + arch + ".tar.gz"
		},
		Lookup: func(ctx context.Context, url string) (string, error) {
			list, err := web.Read(ctx, m, goReleases)
			if err != nil {
				return "", err
			}
			return goSum(list, path.Base(url))
		},
	}}
}
