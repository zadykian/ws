package otelcol

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// version is otelcol-contrib's, from the .deb of the collector's release. sums are the SHA-256 of
// that .deb for each architecture, as the release's .deb.sha256 files give it.
const version = "0.162.0"

var sums = map[string]string{
	"amd64": "0b2b37eeb83db8e19a5f9dfc60894d058c739384a226ac085e674561befaad44",
	"arm64": "b6163e6639989ac608ae2ba58b891128f2f31bba1b0af31a937ac5d0de7b0c23",
}

// downloadStep is the name of the step that fetches the .deb.
const downloadStep = "Collector download"

const releases = "https://github.com/open-telemetry/opentelemetry-collector-releases/releases"

// debName and debURL name the release's .deb for an architecture.
func debName(arch string) string {
	return fmt.Sprintf("otelcol-contrib_%s_linux_%s.deb", version, arch)
}
func debURL(arch string) string  { return releases + "/download/v" + version + "/" + debName(arch) }
func debPath(arch string) string { return "/var/cache/" + debName(arch) }

// installed reports whether the pinned version is installed.
func installed(ctx context.Context, m effect.Machine) (bool, error) {
	out, err := m.Run(ctx, "dpkg-query", "-W", "-f", "${Version} ${db:Status-Abbrev}\n", "otelcol-contrib")
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
		return "", fmt.Errorf("otelcol-contrib has no release for the architecture %s", arch)
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
		return []host.Resource{host.Download{M: m, URL: debURL(arch), Path: debPath(arch), SHA256: sums[arch], Mode: 0o644}}, nil
	}
}
