package zed

import (
	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// serversDir is where the npm servers install. The steps and the settings both name it.
func serversDir(home string) string { return home + "/.local/lib/zed-servers" }

// serverFiles returns the directories and the two files of the lock.
// ~/.local and ~/.local/lib are made only where missing, as root's would break the user's own
// tools, such as claude and pip --user.
func serverFiles(m effect.Machine, user host.User, uid, gid int) ([]host.Resource, error) {
	dir := serversDir(user.Home)
	parts := []host.Resource{
		host.Dir{M: m, Path: user.Home + "/.local", Mode: 0o755, UID: uid, GID: gid, CreateOnly: true},
		host.Dir{M: m, Path: user.Home + "/.local/lib", Mode: 0o755, UID: uid, GID: gid, CreateOnly: true},
		host.Dir{M: m, Path: dir, Mode: 0o755, UID: uid, GID: gid},
	}
	for _, name := range []string{"package.json", "package-lock.json"} {
		content, err := serversLock.ReadFile("servers/" + name)
		if err != nil {
			return nil, err
		}
		parts = append(parts, host.File{M: m, Path: dir + "/" + name, Content: content, Mode: 0o644,
			UID: uid, GID: gid})
	}
	return parts, nil
}
