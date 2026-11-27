// Package update is ws update. It replaces the file ws is installed in with a release's binary
// and changes nothing else. It runs no curl and no shell.
package update

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/zadykian/ws/engine/core/check"
	"github.com/zadykian/ws/engine/core/web"
)

// latest returns the tag that the redirect of releases/latest names. It reads the redirect rather
// than GitHub's API, which allows 60 calls an hour without a token.
func latest(ctx context.Context, releases string) (string, error) {
	address := releases + "/latest"
	client := &http.Client{
		Transport:     web.Transport,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return "", fmt.Errorf("finding the latest release: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("finding the latest release: %w", err)
	}
	_ = resp.Body.Close() // only the redirect matters
	location, err := resp.Location()
	if err != nil {
		return "", fmt.Errorf("finding the latest release at %s: %s", address, resp.Status)
	}
	tag := path.Base(location.Path)
	if !check.Release(tag) {
		return "", fmt.Errorf("%s leads to %q, not to a release", address, tag)
	}
	return tag, nil
}

// get returns the file at address. It follows a redirect to HTTPS alone.
func get(ctx context.Context, address string) ([]byte, error) {
	client := web.Client()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", address, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", address, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("downloading %s: %s", address, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("downloading %s: %w", address, err)
	}
	return body, nil
}

// checksum returns the SHA-256 that sums, a ws.sha256 as sha256sum writes it, gives the file name:
// the first of its lines "HASH  NAME" or "HASH *NAME".
func checksum(sums, name string) (string, bool) {
	for line := range strings.SplitSeq(sums, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == name {
			return fields[0], true
		}
	}
	return "", false
}
