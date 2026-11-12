package devtools

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"runtime"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// Where Go is installed, and where its tarball is downloaded and unpacked before the tree is moved
// into place. Both go under /usr/local/src: root's, and on disk where /tmp may be in memory.
// Neither path names the version, so a run removes what any earlier run left. The unpacked tree's
// does, so a tree cut short for another version never mixes with this one.
const (
	goDir     = "/usr/local/go"
	goTarball = "/usr/local/src/go-install.tar.gz"
	goStaging = "/usr/local/src/go-install"
)

// Go is Go from go.dev's tarball, as Go's install guide has it, in /usr/local/go, whose bin the
// shell role's .profile puts on PATH.
type Go struct {
	M effect.Machine
	// Version is the Go to install. Empty means GoVersion.
	Version string
	// Arch is Go's name for the machine's architecture. Empty means runtime.GOARCH.
	Arch string
	// unpack extracts the tarball. Nil means host.UnpackFile.
	unpack func(archive, dir string, strip int) error
}

func (g Go) version() string {
	if g.Version == "" {
		return GoVersion
	}
	return g.Version
}

func (g Go) arch() string {
	if g.Arch == "" {
		return runtime.GOARCH
	}
	return g.Arch
}

// installed returns the first line of VERSION, such as go1.27.3, or empty where there is no Go yet.
func (g Go) installed() (string, error) {
	data, err := g.M.ReadFile(goDir + "/VERSION")
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	first, _, _ := strings.Cut(string(data), "\n")
	return first, nil
}

// goTree is Go's tarball of the version, from go.dev, unpacked in /usr/local/go. dpkg's name of the
// architecture is Go's. go.dev's list gives the SHA-256, so a plan that finds the tree current
// fetches nothing.
func goTree(m effect.Machine, version string) host.Tree {
	return host.Tree{M: m, Path: goDir, Strip: 1, Source: host.PinnedFile{
		M: m, Version: version,
		URL: func(version, arch string) string {
			return "https://go.dev/dl/go" + version + ".linux-" + arch + ".tar.gz"
		},
		Lookup: func(ctx context.Context, url string) (string, error) {
			list, err := fetch(ctx, m, goReleases)
			if err != nil {
				return "", err
			}
			return goSum(list, path.Base(url))
		},
	}}
}
