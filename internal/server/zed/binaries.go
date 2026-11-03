package zed

import (
	"context"
	"fmt"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// A release pins one server's version and the SHA-256 of its binary for each GOARCH.
type release struct {
	version string
	sha256  map[string]string
}

// digest is r's SHA-256 for goarch. A GOARCH with no entry is an error, so Steps asks for the
// digest before it builds a URL.
func (r release) digest(goarch string) (string, error) {
	sum, ok := r.sha256[goarch]
	if !ok {
		return "", fmt.Errorf("no pin for GOARCH %q", goarch)
	}
	return sum, nil
}

// pgls is postgres-language-server, SQL's language server for Zed, from its release's binary.
// The SHA-256 is GitHub's asset digest, as the release has no checksum file. For another version:
//
//	gh api repos/supabase-community/postgres-language-server/releases/tags/TAG \
//	  --jq '.assets[] | {name, digest}'
var pgls = release{
	version: "0.28.2",
	sha256: map[string]string{
		"amd64": "82ff82e39be65824378afa5d02626a1b42e8d8e48f97cb4938e4d477c8a9be52",
		"arm64": "7be7de3660df047fd712f8fca4d6c42ecf1515daad952715eb02e29053ce256c",
	},
}

// dockerLS is docker-language-server, Dockerfile's and Compose's language server for Zed,
// likewise, with repos/docker/docker-language-server in that command.
var dockerLS = release{
	version: "0.20.1",
	sha256: map[string]string{
		"amd64": "01907aa5b0eae11e44cffea0a993d08aa155542a9af570295dd1dff39e67692a",
		"arm64": "bd56c7815e0a22cfb708669f3d5e817de91d9b54039ff7e52867142a132ad8d7",
	},
}

// pglsURL is r's asset for goarch. The asset spells the architectures x86_64 and aarch64.
func pglsURL(r release, goarch string) string {
	target := map[string]string{"amd64": "x86_64", "arm64": "aarch64"}[goarch]
	return "https://github.com/supabase-community/postgres-language-server/releases/download/" +
		r.version + "/postgres-language-server_" + target + "-unknown-linux-gnu"
}

// dockerLSURL is r's asset for goarch, under the tag "v" and the version.
func dockerLSURL(r release, goarch string) string {
	return "https://github.com/docker/docker-language-server/releases/download/v" + r.version +
		"/docker-language-server-linux-" + goarch + "-v" + r.version
}

// download is the step that puts r's binary at /usr/local/bin/name, root's and mode 0755 so any
// ws_user can run it. A goarch with no pin fails the step's Plan, before any URL exists.
func download(m effect.Machine, goarch, name string, r release, url func(release, string) string) host.Lazy {
	return host.Lazy{Title: "Zed " + name, Parts: func(context.Context) ([]host.Resource, error) {
		sum, err := r.digest(goarch)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		return []host.Resource{host.Download{M: m, URL: url(r, goarch), Path: "/usr/local/bin/" + name,
			SHA256: sum, Mode: 0o755}}, nil
	}}
}
