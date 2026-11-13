package docker

import (
	"embed"

	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

//go:embed files/docker.slice.tmpl files/resolved.conf.tmpl files/override.conf
var files embed.FS

// Slice returns docker.slice, whose limits are over dockerd and every container together. The
// values must hold the limits, which Steps works out where they are empty.
func Slice(d values.Docker) ([]byte, error) {
	return host.Render(files, "files/docker.slice.tmpl", d)
}

// Resolved returns the drop-in that makes systemd-resolved listen on the default bridge's address.
func Resolved(d values.Docker) ([]byte, error) {
	return host.Render(files, "files/resolved.conf.tmpl", d)
}

// Override returns docker.service's drop-in, which puts dockerd in docker.slice. It replaces the
// limits that systemctl edit writes. dockerd moves into the slice as it restarts, and each running
// container as it next starts.
func Override() ([]byte, error) { return files.ReadFile("files/override.conf") }
