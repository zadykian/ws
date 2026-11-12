package fresh

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// version is the .deb's, and tag the release that holds it. sums are the SHA-256 of that .deb for
// each architecture, as the release's assets give it.
const (
	version = "0.5.7-1"
	tag     = "v0.5.7"
)

var sums = map[string]string{
	"amd64": "d98b2cac0d46dbc1c8111eed15fb56c956a338d95740e07fd3ddae72ad397523",
	"arm64": "fb5472fb8c24369b3b191a6dd9cda9239483df4f5a95e2180b991b38c34d1eb9",
}

const releases = "https://github.com/sinelaw/fresh/releases"

// debName and debURL name the release's .deb for an architecture.
func debName(arch string) string   { return fmt.Sprintf("fresh-editor_%s_%s.deb", version, arch) }
func debURL(_, arch string) string { return releases + "/download/" + tag + "/" + debName(arch) }
func debPath(arch string) string   { return path.Join(host.CacheDir, debName(arch)) }

// installed reports whether the pinned version is installed.
func installed(ctx context.Context, m effect.Machine) (bool, error) {
	out, err := m.Run(ctx, "dpkg-query", "-W", "-f", "${Version} ${db:Status-Abbrev}\n", "fresh-editor")
	// dpkg-query exits with 1 for a package it does not know.
	var ee *exec.ExitError
	if err != nil && (!errors.As(err, &ee) || ee.ExitCode() != 1) {
		return false, err
	}
	return strings.TrimSpace(string(out)) == version+" ii", nil
}

func architecture(ctx context.Context, m effect.Machine) (string, error) {
	out, err := m.Run(ctx, "dpkg", "--print-architecture")
	if err != nil {
		return "", fmt.Errorf("reading dpkg's architecture: %w", err)
	}
	arch := strings.TrimSpace(string(out))
	if sums[arch] == "" {
		return "", fmt.Errorf("fresh-editor has no release for the architecture %s", arch)
	}
	return arch, nil
}

// download makes the part that fetches the release's .deb, checked against its SHA-256 before dpkg
// reads it, where the version is not installed.
func download(m effect.Machine) func(context.Context) ([]host.Resource, error) {
	return func(ctx context.Context) ([]host.Resource, error) {
		if ok, err := installed(ctx, m); err != nil || ok {
			return nil, err
		}
		arch, err := architecture(ctx, m)
		if err != nil {
			return nil, err
		}
		return []host.Resource{host.Dir{M: m, Path: host.CacheDir, Mode: 0o755}, host.Download{M: m, URL: debURL(version, arch), Path: debPath(arch), SHA256: sums[arch], Mode: 0o644}}, nil
	}
}
