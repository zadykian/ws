package ikeward

import (
	"context"
	"fmt"
	"strings"
	"time"

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

// downloadTries and downloadWait are the role's: five tries, ten seconds apart, as a server can fail
// for a moment. A test shortens the wait.
const downloadTries = 5

var downloadWait = 10 * time.Second

// retrying is a machine whose Download tries again after a failure.
type retrying struct {
	effect.Machine
	wait time.Duration
}

func (r retrying) Download(ctx context.Context, url, path, sum string, a effect.Attrs) error {
	var err error
	for try := 1; try <= downloadTries; try++ {
		if err = r.Machine.Download(ctx, url, path, sum, a); err == nil {
			return nil
		}
		if try == downloadTries {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(r.wait):
		}
	}
	return err
}

// download makes the release's binary, where install.sh would put it, without install.sh. Download
// checks it against the SHA-256 before it writes the file, and fetches again where the file holds
// another, a former version's.
func download(m effect.Machine) func(context.Context) ([]host.Resource, error) {
	return func(ctx context.Context) ([]host.Resource, error) {
		arch, err := architecture(ctx, m)
		if err != nil {
			return nil, err
		}
		return []host.Resource{host.Download{M: retrying{m, downloadWait}, SHA256: sums[arch], Path: binary, Mode: 0o755,
			URL: releases + "/download/v" + version + "/ikeward-linux-" + arch}}, nil
	}
}
