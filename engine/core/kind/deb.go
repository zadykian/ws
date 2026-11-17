package kind

import (
	"context"
	"errors"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/apt"
	"github.com/zadykian/ws/engine/core/resource/fetch"
)

// debSpec is a unit of the kind deb: a .deb pinned by its version and by the SHA-256 of each
// architecture's file, as dpkg names the architecture. {version} and {arch} in URL stand for
// those. The file waits in fetch.CacheDir, checked against its sum, and its own package and version
// are checked before dpkg sees it.
type debSpec struct {
	base
	Package string            `toml:"package"`
	Version string            `toml:"version"`
	URL     string            `toml:"url"`
	SHA256  map[string]string `toml:"sha256"`
	// Stop is a service that the package starts, kept stopped and disabled where it installs, for
	// a later unit to set up first.
	Stop string `toml:"stop"`
}

func (s debSpec) parts(e Env, _ fs.FS) (func(context.Context) ([]resource.Resource, error), error) {
	if s.Package == "" || s.Version == "" || s.URL == "" || len(s.SHA256) == 0 {
		return nil, errors.New("a deb takes a package, a version, a url and a sha256 for each architecture")
	}
	for arch, sum := range s.SHA256 {
		if len(sum) != 64 || strings.Trim(sum, "0123456789abcdef") != "" {
			return nil, errors.New("sha256 of " + arch + " is not 64 lowercase hex digits")
		}
	}
	if s.Stop != "" {
		if err := effect.CheckUnitName("stop", s.Stop); err != nil {
			return nil, err
		}
	}
	url := func(version, arch string) string {
		return strings.NewReplacer("{version}", version, "{arch}", arch).Replace(s.URL)
	}
	pkg := apt.Package{M: e.M, Name: s.Package, Source: fetch.PinnedFile{M: e.M, Version: s.Version, URL: url, SHA256: s.SHA256}}
	var r resource.Resource = pkg
	if s.Stop != "" {
		r = held{pkg: pkg, m: e.M, u: e.U, out: e.Out, unit: s.Stop}
	}
	return func(context.Context) ([]resource.Resource, error) { return []resource.Resource{r}, nil }, nil
}
