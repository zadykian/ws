package kind

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"strings"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/fetch"
)

// binarySpec is a unit of the kind binary: a release's file, root's at Path. Its version and the
// SHA-256 of each architecture's file are pinned, the architecture as dpkg names it. The key "all"
// is for a file that every architecture shares. {version} and {arch} in URL stand for those. A
// download makes the file only where its sum matches, and again where the file holds another.
type binarySpec struct {
	base
	Version string            `toml:"version"`
	URL     string            `toml:"url"`
	SHA256  map[string]string `toml:"sha256"`
	// ArchNames spells an architecture as the release's assets do, dpkg's name to theirs, where
	// they differ: amd64 = "x86_64". An architecture left out keeps dpkg's name.
	ArchNames map[string]string `toml:"arch_names"`
	Path      string            `toml:"path"`
	// Mode is the file's permission bits, 0o755 where unset.
	Mode uint32 `toml:"mode"`
}

func (s binarySpec) parts(e Env, _ fs.FS) (func(context.Context) ([]resource.Resource, error), error) {
	switch {
	case s.Version == "" || s.URL == "" || len(s.SHA256) == 0:
		return nil, errors.New("a binary takes a version, a url and a sha256 for each architecture")
	case !path.IsAbs(s.Path) || path.Clean(s.Path) != s.Path || s.Path == "/":
		return nil, errors.New("path " + s.Path + " is not a clean absolute path of a file")
	case s.Mode > 0o777:
		return nil, errors.New("mode is not a file's permission bits")
	}
	if err := checkURL(s.URL); err != nil {
		return nil, err
	}
	if err := checkSums(s.SHA256); err != nil {
		return nil, err
	}
	mode := fs.FileMode(s.Mode)
	if s.Mode == 0 {
		mode = 0o755
	}
	url := func(version, arch string) string {
		if name, ok := s.ArchNames[arch]; ok {
			arch = name
		}
		return strings.NewReplacer("{version}", version, "{arch}", arch).Replace(s.URL)
	}
	return func(context.Context) ([]resource.Resource, error) {
		return []resource.Resource{fetch.PinnedFile{M: e.M, Version: s.Version, URL: url, SHA256: s.SHA256, Path: s.Path, Mode: mode}}, nil
	}, nil
}
