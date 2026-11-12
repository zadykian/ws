package fresh

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
)

// pinned is the editor's .deb as a package, which plans from the pin and removes the file once it
// installs.
func pinned(m effect.Machine) host.Package {
	return host.Package{M: m, Name: "fresh-editor", Source: host.PinnedFile{M: m, Version: version, URL: debURL, SHA256: sums}}
}
