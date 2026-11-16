package otelcol

import (
	"embed"
	"strings"

	"github.com/zadykian/ws/engine/core/resource"
)

//go:embed files
var files embed.FS

type templateData struct {
	Endpoint string
	// Remote is set where an endpoint is given, and Insecure where the endpoint is http, whose gRPC
	// runs without TLS.
	Remote, Insecure bool
}

func data(endpoint string) templateData {
	return templateData{endpoint, endpoint != "", strings.HasPrefix(endpoint, "http://")}
}

// Config returns the collector's config. The remote collector's endpoint is not in it, as the
// service's environment gives it; an empty endpoint leaves claude's pipeline out.
func Config(endpoint string) ([]byte, error) {
	return resource.Render(files, "files/config.yaml.tmpl", data(endpoint))
}

// Env returns the file with the endpoint for the service, which only root reads.
func Env(endpoint string) ([]byte, error) {
	return resource.Render(files, "files/claude.env.tmpl", data(endpoint))
}
