package rootless

import "embed"

//go:embed files/rootless.conf files/delegate.conf
var unitFiles embed.FS

// delegatePath is where user@.service's drop-in goes.
const delegatePath = "/etc/systemd/system/user@.service.d/delegate.conf"

// overrideDropIn returns the rootless unit's drop-in. It puts the unit in the user's docker.slice
// and pins the drivers, so that a package installed later changes neither.
func overrideDropIn() ([]byte, error) { return unitFiles.ReadFile("files/rootless.conf") }

// delegateDropIn returns user@.service's drop-in, which gives a user's systemd the io controller.
// Ubuntu 26.04 delegates cpu, memory and pids by default, and a container's --device-read-bps needs
// io.
func delegateDropIn() ([]byte, error) { return unitFiles.ReadFile("files/delegate.conf") }
