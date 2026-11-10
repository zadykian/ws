package fans

import (
	"context"
	"errors"
	"io/fs"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

const (
	sourceDir = "/usr/src/nct6687d-" + nct6687Short
	tarball   = "/var/cache/nct6687d-" + nct6687Short + ".tar.gz"
)

// source puts the driver's source in /usr/src for DKMS: GitHub's tarball of the commit, checked
// against its SHA-256 before the step unpacks it. The tarball goes after.
type source struct {
	m effect.Machine
	// sum is the tarball's SHA-256, in hex.
	sum string
	// unpack extracts the tarball. Nil means host.UnpackFile.
	unpack func(archive, dir string, strip int) error
}

// Plan downloads and unpacks the source where nct6687.c is not in place.
func (s source) Plan(ctx context.Context) ([]host.Change, error) {
	if _, err := s.m.Stat(sourceDir + "/nct6687.c"); err == nil {
		return nil, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	fetch, err := host.Download{
		M: s.m, URL: "https://github.com/Fred78290/nct6687d/archive/" + nct6687Commit + ".tar.gz",
		Path: tarball, SHA256: s.sum, Mode: 0o644,
	}.Plan(ctx)
	if err != nil {
		return nil, err
	}
	return append(fetch, unpackSource{s}), nil
}

// unpackSource unpacks the tarball into the source directory, dropping the tree's own top
// directory, and removes the tarball.
type unpackSource struct{ s source }

func (unpackSource) String() string { return "unpack the driver's source in " + sourceDir }

func (c unpackSource) Apply(ctx context.Context) error {
	unpack := c.s.unpack
	if unpack == nil {
		unpack = host.UnpackFile
	}
	if err := unpack(tarball, sourceDir, 1); err != nil {
		return err
	}
	_, err := c.s.m.RunRoot(ctx, "rm", "-f", "--", tarball)
	return err
}
