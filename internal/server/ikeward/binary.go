package ikeward

import (
	"context"
	"fmt"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// version is ikeward's, from its release's binary. sums are the SHA-256 of that binary for each
// architecture, as the release's ikeward.sha256 gives it.
const version = "0.3.0"

var sums = map[string]string{
	"amd64": "b78ba9456f246ddaa5f08f34706cf96b975ad3717a69f142cbcbd893294b2d23",
	"arm64": "711b9a1c72b59ce7bc453d94c790855588b529803b8789abc72ed3b405e83e3b",
}

// binary is where install puts ikeward, and where its services, the routing's and strongSwan run
// it: install refuses to run as any other program.
const binary = "/usr/local/sbin/ikeward"

const releases = "https://github.com/zadykian/ikeward/releases"

// architecture returns dpkg's name for the machine's, which the release's assets use.
func architecture(ctx context.Context, m effect.Machine) (string, error) {
	out, err := m.Run(ctx, "dpkg", "--print-architecture")
	if err != nil {
		return "", fmt.Errorf("reading dpkg's architecture: %w", err)
	}
	arch := strings.TrimSpace(string(out))
	if sums[arch] == "" {
		return "", fmt.Errorf("ikeward has no release for the architecture %s", arch)
	}
	return arch, nil
}

// download makes the release's binary without install.sh, where install.sh would put it. Download
// checks it against the SHA-256 before it writes the file, and fetches again where the file holds
// another, a former version's.
func download(m effect.Machine) func(context.Context) ([]host.Resource, error) {
	return func(ctx context.Context) ([]host.Resource, error) {
		arch, err := architecture(ctx, m)
		if err != nil {
			return nil, err
		}
		return []host.Resource{host.Download{M: m, SHA256: sums[arch], Path: binary, Mode: 0o755,
			URL: releases + "/download/v" + version + "/ikeward-linux-" + arch}}, nil
	}
}
