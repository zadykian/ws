package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/fetch"
	"github.com/zadykian/ws/engine/core/host/file"
)

// keyURL and ascSHA256 are Docker's apt key as its install guide for Ubuntu gives it. The checksum
// is that of the key with fingerprint `9DC8 5822 9FC7 DD38 854A E2D8 8D81 803C 0EBF CD88`, so that
// a changed key fails the run rather than being trusted unseen.
const (
	keyPath = "/etc/apt/keyrings/docker.asc"
	keyURL  = "https://download.docker.com/linux/ubuntu/gpg"
)

// ascSHA256 is a variable so that a test can pin the key it serves.
var ascSHA256 = "1500c1f56fa9e26b9b8f42452a553675796ade0807cdce11975eb98170b3a570"

// sourcesPath is the file that names Docker's repository.
const sourcesPath = "/etc/apt/sources.list.d/docker.sources"

// Sources returns the deb822 source of Docker's repository for a release and an architecture.
func Sources(release, arch string) []byte {
	return fmt.Appendf(nil, `Types: deb
URIs: https://download.docker.com/linux/ubuntu
Suites: %s
Components: stable
Architectures: %s
Signed-By: %s
`, release, arch, keyPath)
}

// release reads the Ubuntu release's codename from os-release.
func release(m effect.Machine) (string, error) {
	data, err := m.ReadFile("/etc/os-release")
	if err != nil {
		return "", err
	}
	for line := range strings.Lines(string(data)) {
		if v, ok := strings.CutPrefix(line, "VERSION_CODENAME="); ok {
			return strings.Trim(strings.TrimSpace(v), `"`), nil
		}
	}
	return "", fmt.Errorf("/etc/os-release has no VERSION_CODENAME")
}

// repository makes the parts that add Docker's apt key and source. It reads the release and
// dpkg's architecture as it plans.
func repository(m effect.Machine) func(context.Context) ([]host.Resource, error) {
	return func(ctx context.Context) ([]host.Resource, error) {
		rel, err := release(m)
		if err != nil {
			return nil, err
		}
		arch, err := fetch.Architecture(ctx, m)
		if err != nil {
			return nil, err
		}
		return []host.Resource{
			fetch.Download{M: m, URL: keyURL, Path: keyPath, SHA256: ascSHA256, Mode: 0o644},
			file.File{M: m, Path: sourcesPath, Content: Sources(rel, arch), Mode: 0o644},
		}, nil
	}
}
