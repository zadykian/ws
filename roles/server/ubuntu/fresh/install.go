package fresh

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/apt"
)

// pinned is the editor's .deb as a package, which plans from the pin and removes the file once it
// installs.
func pinned(m effect.Machine) apt.Package {
	return apt.Package{M: m, Name: "fresh-editor", Source: host.PinnedFile{M: m, Version: version, URL: debURL, SHA256: sums}}
}
