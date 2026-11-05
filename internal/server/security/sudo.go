package security

import (
	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

const noCachePath = "/etc/sudoers.d/ws-no-cache"

// sudoParts makes the step's parts: the drop-in that ends sudo's cache. visudo reads it before it
// is in place, though a broken one would not lock anyone out: sudo-rs skips the file with an error.
// sudo skips a name with a dot, so the file under check has no effect.
func sudoParts(m effect.Machine) ([]host.Resource, error) {
	conf, err := files.ReadFile("files/ws-no-cache")
	if err != nil {
		return nil, err
	}
	return []host.Resource{host.Validated{
		File: host.File{M: m, Path: noCachePath, Content: conf, Mode: 0o440},
		Check: func(path string) (string, []string) {
			return "visudo", []string{"-cf", path}
		},
	}}, nil
}
