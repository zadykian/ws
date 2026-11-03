package devtools

import (
	"errors"
	"io/fs"
	"runtime"
	"strings"

	"github.com/zadykian/ws/effect"
)

// Where Go is installed, and where its tarball is downloaded and unpacked before the tree is moved
// into place, under /usr/local/src: root's, and on disk where /tmp may be in memory. Neither path
// names the version, so a run removes what any earlier run left; the unpacked tree's does, so a
// tree cut short for another version never mixes with this one.
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
