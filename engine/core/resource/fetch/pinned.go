package fetch

import (
	"context"
	"fmt"
	"io/fs"
	"path"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// CacheDir is where the pinned file of a package waits for its install.
const CacheDir = "/var/cache/ws"

// PinnedFile is a file of a release, root's at Path. The role pins its Version and the SHA-256 of
// each architecture's file. The machine's architecture, as dpkg names it, picks the URL and the
// sum. The key "all" in SHA256 is for a file that every architecture shares. It reads no
// architecture, and URL gets "all".
type PinnedFile struct {
	M       effect.Machine
	Version string
	URL     func(version, arch string) string
	SHA256  map[string]string
	Path    string
	Mode    fs.FileMode
}

// resolve reads the architecture and returns the URL of the file, and the sum that SHA256 pins
// for it. Without a pin it fails before a URL exists.
func (p PinnedFile) resolve(ctx context.Context) (url, sum string, err error) {
	arch := "all"
	if _, shared := p.SHA256[arch]; !shared {
		if arch, err = Architecture(ctx, p.M); err != nil {
			return "", "", err
		}
	}
	sum, pinned := p.SHA256[arch]
	if !pinned {
		return "", "", fmt.Errorf("no SHA-256 pinned for the architecture %s", arch)
	}
	return p.URL(p.Version, arch), sum, nil
}

// download returns the Download that holds the pin.
func (p PinnedFile) download(ctx context.Context) (Download, error) {
	url, sum, err := p.resolve(ctx)
	if err != nil {
		return Download{}, err
	}
	return Download{M: p.M, URL: url, Path: p.Path, SHA256: sum, Mode: p.Mode}, nil
}

// Plan plans what Download plans for the file of the machine's architecture.
func (p PinnedFile) Plan(ctx context.Context) ([]resource.Change, error) {
	d, err := p.download(ctx)
	if err != nil {
		return nil, err
	}
	return d.Plan(ctx)
}

// Report reports what Download reports for the file of the machine's architecture.
func (p PinnedFile) Report(ctx context.Context) ([]step.Item, error) {
	d, err := p.download(ctx)
	if err != nil {
		return nil, err
	}
	return d.Report(ctx)
}

// Cached is download for a package's source: its file waits in CacheDir, named by its URL.
func (p PinnedFile) Cached(ctx context.Context) (Download, error) {
	return p.cachedIn(ctx, CacheDir)
}

// cachedIn is Cached with the file waiting in dir.
func (p PinnedFile) cachedIn(ctx context.Context, dir string) (Download, error) {
	p.Path = dir
	d, err := p.download(ctx)
	if err != nil {
		return Download{}, err
	}
	d.Path, d.Mode = path.Join(dir, path.Base(d.URL)), 0o644
	return d, nil
}
