package network

import (
	"embed"
	"encoding/json"

	"github.com/zadykian/ws/engine/core/host"
)

//go:embed files
var files embed.FS

// netplanPath is the direct link's netplan file. Its name sorts after the installer's.
const netplanPath = "/etc/netplan/60-direct-link.yaml"

// Render returns the direct link's netplan file. A JSON string is a YAML string too, so the
// interface and the address are quoted as JSON quotes them, whatever they hold.
func Render(iface, address string) ([]byte, error) {
	quoted := func(s string) string {
		b, _ := json.Marshal(s) // a string always marshals
		return string(b)
	}
	return host.Render(files, "files/60-direct-link.yaml.tmpl", struct{ Interface, Address string }{quoted(iface), quoted(address)})
}
