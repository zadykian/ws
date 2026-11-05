package docker

import (
	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// rootless sets up ws_user's own daemon. Root writes the user's files, whose links it refuses.
func rootless(v values.Values, m effect.Machine, size DiskSize) step.Step {
	user := host.User{Name: v.User, Home: v.Home}
	return user.Owned(m, rootlessStep, configStep, func(uid, gid int) ([]host.Resource, error) {
		d, err := WithLimits(m, v.Docker)
		if err != nil {
			return nil, err
		}
		cache, err := cacheLimits(m, d, size)
		if err != nil {
			return nil, err
		}
		daemonJSON, err := RootlessJSON(d, cache)
		if err != nil {
			return nil, err
		}
		slice, err := Slice(d)
		if err != nil {
			return nil, err
		}
		override, err := RootlessOverride()
		if err != nil {
			return nil, err
		}
		conf, units := v.Home+"/.config/docker", v.Home+"/.config/systemd/user"
		dropIn := units + "/docker.service.d"
		daemon := host.Validated{
			File: host.File{M: m, Path: conf + "/daemon.json", Content: daemonJSON, Mode: 0o644, UID: uid, GID: gid},
			Check: func(path string) (string, []string) {
				return "dockerd", []string{"--validate", "--config-file", path}
			},
		}
		sliceFile := host.File{M: m, Path: units + "/docker.slice", Content: slice, Mode: 0o644, UID: uid, GID: gid}
		service := host.File{M: m, Path: dropIn + "/ws.conf", Content: override, Mode: 0o644, UID: uid, GID: gid}
		return []host.Resource{
			host.Plain{M: m, Home: v.Home, Paths: []string{daemon.Path, sliceFile.Path, service.Path}},
			subIDs{m, v.User},
			host.Dir{M: m, Path: v.Home + "/.config", Mode: 0o755, UID: uid, GID: gid, CreateOnly: true},
			host.Dir{M: m, Path: conf, Mode: 0o755, UID: uid, GID: gid},
			host.Dir{M: m, Path: v.Home + "/.config/systemd", Mode: 0o755, UID: uid, GID: gid},
			host.Dir{M: m, Path: units, Mode: 0o755, UID: uid, GID: gid},
			host.Dir{M: m, Path: dropIn, Mode: 0o755, UID: uid, GID: gid},
			daemon,
			sliceFile,
			service,
		}, nil
	})
}
