package fresh

import (
	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	editorfresh "github.com/zadykian/ws/internal/editor/fresh"
	"github.com/zadykian/ws/internal/server/devtools"
	"github.com/zadykian/ws/internal/server/values"
)

// configFiles returns ~/.config/fresh with config.json and the theme, which ws replaces whole. The
// directory stays the user's, as Fresh writes its own files there. The old file stays beside the new
// one as a backup: Fresh's scan of themes/ takes only .json, so a backup isn't a theme.
func configFiles(m effect.Machine, v values.Values, uid, gid int) ([]host.Resource, error) {
	config, err := editorfresh.Config(devtools.ServerPaths(v.Home))
	if err != nil {
		return nil, err
	}
	dir := v.Home + "/.config/fresh"
	themes := dir + "/themes"
	theme := themes + "/jetbrains-dark.json"
	return []host.Resource{
		host.Plain{M: m, Home: v.Home, Paths: []string{dir + "/config.json", theme}},
		host.Dir{M: m, Path: v.Home + "/.config", Mode: 0o755, UID: uid, GID: gid, CreateOnly: true},
		host.Dir{M: m, Path: dir, Mode: 0o755, UID: uid, GID: gid},
		host.Dir{M: m, Path: themes, Mode: 0o755, UID: uid, GID: gid},
		host.File{M: m, Path: dir + "/config.json", Content: config, Mode: 0o644, UID: uid, GID: gid, Backup: true},
		host.File{M: m, Path: theme, Content: editorfresh.Theme, Mode: 0o644, UID: uid, GID: gid, Backup: true},
	}, nil
}
