package fresh

import "fmt"

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
