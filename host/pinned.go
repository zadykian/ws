package host

import (
	"context"
	"fmt"
	"io/fs"
	"path"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
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

// download reads the architecture and returns the Download that holds the pin. A missing sum
// fails before a URL exists.
func (p PinnedFile) download(ctx context.Context) (Download, error) {
	arch := "all"
	if _, shared := p.SHA256[arch]; !shared {
		var err error
		if arch, err = Architecture(ctx, p.M); err != nil {
			return Download{}, err
		}
	}
	sum, ok := p.SHA256[arch]
	if !ok {
		return Download{}, fmt.Errorf("no SHA-256 pinned for the architecture %s", arch)
	}
	return Download{M: p.M, URL: p.URL(p.Version, arch), Path: p.Path, SHA256: sum, Mode: p.Mode}, nil
}

// Plan plans what Download plans for the file of the machine's architecture.
func (p PinnedFile) Plan(ctx context.Context) ([]Change, error) {
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

// cached is download for a package's source: its file waits in CacheDir, named by its URL.
func (p PinnedFile) cached(ctx context.Context) (Download, error) {
	p.Path = CacheDir
	d, err := p.download(ctx)
	if err != nil {
		return Download{}, err
	}
	d.Path, d.Mode = path.Join(CacheDir, path.Base(d.URL)), 0o644
	return d, nil
}
