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

// treeSpec is a unit of the kind tree: a directory, root's at Path, unpacked from pinned archives
// in their order. A later archive's file replaces an earlier one's, and a new pin replaces the
// directory whole.
type treeSpec struct {
	base
	Path string `toml:"path"`
	// Strip is the leading parts of each name dropped as an archive unpacks.
	Strip int `toml:"strip"`
	// ArchNames spells an architecture as the archives' URLs do, the machine's name to theirs,
	// where they differ: amd64 = "x64".
	ArchNames map[string]string `toml:"arch_names"`
	Archives  []treeArchive     `toml:"archive"`
}

// treeArchive is one archive of a tree: its version and the SHA-256 of each architecture's file,
// the architecture as dpkg names it. {version} and {arch} in URL stand for those.
type treeArchive struct {
	Version string            `toml:"version"`
	URL     string            `toml:"url"`
	SHA256  map[string]string `toml:"sha256"`
}

func (s treeSpec) parts(e Env, _ fs.FS) (func(context.Context) ([]resource.Resource, error), error) {
	switch {
	case !path.IsAbs(s.Path) || path.Clean(s.Path) != s.Path || path.Dir(s.Path) == "/":
		return nil, errors.New("path " + s.Path + " is not a clean absolute path below a directory of /")
	case s.Strip < 0:
		return nil, errors.New("strip is below 0")
	case len(s.Archives) == 0:
		return nil, errors.New("a tree takes an archive at least")
	}
	tree := fetch.Tree{M: e.M, Path: s.Path, Strip: s.Strip}
	for _, a := range s.Archives {
		if a.Version == "" || a.URL == "" || len(a.SHA256) == 0 {
			return nil, errors.New("an archive takes a version, a url and a sha256 for each architecture")
		}
		if err := checkSums(a.SHA256); err != nil {
			return nil, err
		}
		url := func(version, arch string) string {
			if name, ok := s.ArchNames[arch]; ok {
				arch = name
			}
			return strings.NewReplacer("{version}", version, "{arch}", arch).Replace(a.URL)
		}
		tree.Sources = append(tree.Sources, fetch.PinnedFile{M: e.M, Version: a.Version, URL: url, SHA256: a.SHA256})
	}
	return func(context.Context) ([]resource.Resource, error) { return []resource.Resource{tree}, nil }, nil
}
