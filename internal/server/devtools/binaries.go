package devtools

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
)

// A release pins one server's version and the SHA-256 of its binary for each architecture, as
// dpkg names it.
type release struct {
	version string
	sha256  map[string]string
}

// pgls is postgres-language-server, SQL's language server, from its release's binary.
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

// dockerLS is docker-language-server, Dockerfile's and Compose's language server, pinned
// likewise, with repos/docker/docker-language-server in that command.
var dockerLS = release{
	version: "0.20.1",
	sha256: map[string]string{
		"amd64": "01907aa5b0eae11e44cffea0a993d08aa155542a9af570295dd1dff39e67692a",
		"arm64": "bd56c7815e0a22cfb708669f3d5e817de91d9b54039ff7e52867142a132ad8d7",
	},
}

// pglsURL is the asset of the version for the architecture. The asset spells the architectures
// x86_64 and aarch64.
func pglsURL(version, arch string) string {
	target := map[string]string{"amd64": "x86_64", "arm64": "aarch64"}[arch]
	return "https://github.com/supabase-community/postgres-language-server/releases/download/" +
		version + "/postgres-language-server_" + target + "-unknown-linux-gnu"
}

// dockerLSURL is the asset of the version for the architecture, under the tag "v" and the version.
func dockerLSURL(version, arch string) string {
	return "https://github.com/docker/docker-language-server/releases/download/v" + version +
		"/docker-language-server-linux-" + arch + "-v" + version
}

// download is the step that puts r's binary at /usr/local/bin/name, root's and mode 0755 so any
// ws_user can run it. An architecture with no pin fails the step's Plan, before any URL exists.
func download(m effect.Machine, name string, r release, url func(version, arch string) string) host.Group {
	return host.Group{Title: "Devtools " + name, Parts: []host.Resource{host.PinnedFile{M: m,
		Version: r.version, URL: url, SHA256: r.sha256, Path: "/usr/local/bin/" + name, Mode: 0o755}}}
}
