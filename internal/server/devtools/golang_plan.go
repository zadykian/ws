package devtools

import (
	"context"

	"github.com/zadykian/ws/host"
)

// Plan installs Go where the Go in /usr/local/go is another version. A run cut short leaves a
// tarball and an unpacked tree; they go, whatever the version, once Go is installed.
func (g Go) Plan(ctx context.Context) ([]host.Change, error) {
	have, err := g.installed()
	if err != nil {
		return nil, err
	}
	var parts []host.Resource
	var after []host.Change
	if have == "go"+g.version() {
		parts = []host.Resource{host.Absent{M: g.M, Path: goTarball}, host.Absent{M: g.M, Path: goStaging}}
	} else {
		file := "go" + g.version() + ".linux-" + g.arch() + ".tar.gz"
		list, err := fetch(ctx, g.M, goReleases)
		if err != nil {
			return nil, err
		}
		sum, err := goSum(list, file)
		if err != nil {
			return nil, err
		}
		// Download checks the body against the SHA-256 before it writes the tarball, and downloads
		// again where the tarball there has another, another version's.
		parts = []host.Resource{
			host.Download{M: g.M, URL: "https://go.dev/dl/" + file, Path: goTarball, SHA256: sum, Mode: 0o644},
		}
		after = []host.Change{install{g}}
	}
	var changes []host.Change
	for _, part := range parts {
		c, err := part.Plan(ctx)
		if err != nil {
			return nil, err
		}
		changes = append(changes, c...)
	}
	return append(changes, after...), nil
}

// install unpacks the tarball elsewhere and moves the tree into place whole, as tar writes VERSION
// near the start of the tree. An unpack cut short in place would pass for the version at the next
// run.
type install struct{ g Go }

func (c install) String() string { return "install Go " + c.g.version() + " in " + goDir }

func (c install) Apply(ctx context.Context) error {
	unpack := c.g.unpack
	if unpack == nil {
		unpack = host.UnpackFile
	}
	if _, err := c.g.M.RunRoot(ctx, "rm", "-rf", "--", goStaging); err != nil {
		return err
	}
	// The tarball holds the tree under go/, which Unpack drops.
	if err := unpack(goTarball, goStaging, 1); err != nil {
		return err
	}
	for _, args := range [][]string{{"rm", "-rf", "--", goDir}, {"mv", goStaging, goDir}, {"rm", "-f", "--", goTarball}} {
		if _, err := c.g.M.RunRoot(ctx, args[0], args[1:]...); err != nil {
			return err
		}
	}
	return nil
}
