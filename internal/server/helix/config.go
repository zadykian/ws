package helix

import (
	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/values"
)

// configFiles returns ~/.config/helix with config.toml and languages.toml, which ws replaces
// whole. The old file stays beside the new one as a backup.
func configFiles(m effect.Machine, v values.Values, uid, gid int) ([]host.Resource, error) {
	config, err := files.ReadFile("files/config.toml")
	if err != nil {
		return nil, err
	}
	langs, err := languages(v)
	if err != nil {
		return nil, err
	}
	dir := v.Home + "/.config/helix"
	return []host.Resource{
		host.Plain{M: m, Home: v.Home, Paths: []string{dir + "/config.toml", dir + "/languages.toml", dir + "/.ws-grammars"}},
		host.Dir{M: m, Path: v.Home + "/.config", Mode: 0o755, UID: uid, GID: gid, CreateOnly: true},
		host.Dir{M: m, Path: dir, Mode: 0o755, UID: uid, GID: gid},
		host.File{M: m, Path: dir + "/config.toml", Content: config, Mode: 0o644, UID: uid, GID: gid, Backup: true},
		host.File{M: m, Path: dir + "/languages.toml", Content: langs, Mode: 0o644, UID: uid, GID: gid, Backup: true},
	}, nil
}
