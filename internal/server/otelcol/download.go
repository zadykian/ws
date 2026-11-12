package otelcol

import "fmt"

// version is otelcol-contrib's, from the .deb of the collector's release. sums are the SHA-256 of
// that .deb for each architecture, as the release's .deb.sha256 files give it.
const version = "0.162.0"

var sums = map[string]string{
	"amd64": "0b2b37eeb83db8e19a5f9dfc60894d058c739384a226ac085e674561befaad44",
	"arm64": "b6163e6639989ac608ae2ba58b891128f2f31bba1b0af31a937ac5d0de7b0c23",
}

const releases = "https://github.com/open-telemetry/opentelemetry-collector-releases/releases"

// debName and debURL name the release's .deb for an architecture.
func debName(arch string) string {
	return fmt.Sprintf("otelcol-contrib_%s_linux_%s.deb", version, arch)
}
func debURL(_, arch string) string { return releases + "/download/v" + version + "/" + debName(arch) }
