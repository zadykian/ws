package docker

import (
	"embed"

	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/values"
)

//go:embed files
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

// RootlessOverride returns the rootless unit's drop-in. It puts the unit in the user's docker.slice
// and pins the drivers, so that a package installed later changes neither.
func RootlessOverride() ([]byte, error) { return files.ReadFile("files/rootless.conf") }

// Delegate returns user@.service's drop-in, which gives a user's systemd the io controller. Ubuntu
// 26.04 delegates cpu, memory and pids by default, and a container's --device-read-bps needs io.
func Delegate() ([]byte, error) { return files.ReadFile("files/delegate.conf") }
