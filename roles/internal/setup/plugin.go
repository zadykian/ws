package setup

import (
	"path"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
)

const (
	// PluginDir is mise's system directory of plugins on Linux. mise loads a plugin there as it
	// stands, and installs none. Its other system directory, /usr/local/share/mise/plugins, stays
	// empty, and root's own, below its data directory, is where mise installs a plugin.
	PluginDir = "/usr/share/mise/plugins"
	// PluginName is the name of ws's plugin, which hands the units to ws.
	PluginName = "ws"
)

// Plugin returns the resources that link the tree's plugin into pluginDir, which mise loads as a
// system plugin. mise's two directories are made owner's and 755, whatever was there. A
// [bootstrap.plugins] entry would install the plugin again on every run, with a warning.
func Plugin(m effect.Machine, tree, pluginDir string, owner int) []resource.Resource {
	return []resource.Resource{
		file.Dir{M: m, Path: path.Dir(pluginDir), Mode: 0o755, UID: owner, GID: owner},
		file.Dir{M: m, Path: pluginDir, Mode: 0o755, UID: owner, GID: owner},
		file.Link{M: m, Path: path.Join(pluginDir, PluginName), Target: path.Join(tree, "plugins", PluginName)},
	}
}
