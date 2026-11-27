package kind

import (
	"context"
	"errors"
	"fmt"
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
	if err := checkURL(s.URL); err != nil {
		return nil, err
	}
	if err := checkSums(s.SHA256); err != nil {
		return nil, err
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
		r = held{pkg: pkg, u: e.U, unit: s.Stop}
	}
	return func(context.Context) ([]resource.Resource, error) { return []resource.Resource{r}, nil }, nil
}

// checkURL refuses a pin's url that is not https, as the kinds that download share one rule.
func checkURL(url string) error {
	if !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("url %q is no https URL", url)
	}
	return nil
}

// checkSums refuses a pin that is no SHA-256 in lowercase hex, naming its architecture.
func checkSums(sums map[string]string) error {
	for arch, sum := range sums {
		if len(sum) != 64 || strings.Trim(sum, "0123456789abcdef") != "" {
			return errors.New("sha256 of " + arch + " is not 64 lowercase hex digits")
		}
	}
	return nil
}
