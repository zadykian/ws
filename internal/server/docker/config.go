package docker

import (
	"context"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/internal/server/ufw"
	"github.com/zadykian/ws/internal/server/values"
)

// cacheLimits fills the build cache's limits that the values leave empty, from the size of the
// filesystem that holds Docker's data.
func cacheLimits(m effect.Machine, d values.Docker, size DiskSize) (Cache, error) {
	c := Cache{d.BuildCacheReservedSpace, d.BuildCacheMaxUsedSpace, d.BuildCacheMinFreeSpace}
	if c.Reserved == "" || c.MaxUsed == "" || c.MinFree == "" {
		exists := func(p string) bool { _, err := m.Stat(p); return err == nil }
		n, err := dataDiskSize(exists, size)
		if err != nil {
			return c, err
		}
		def := DefaultCache(n)
		c.Reserved, c.MaxUsed, c.MinFree = orDefault(c.Reserved, def.Reserved), orDefault(c.MaxUsed, def.MaxUsed), orDefault(c.MinFree, def.MinFree)
	}
	return c, c.Check(d.BuildCacheKeepDays)
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// configuration makes the parts that configure Docker and the resolver: daemon.json, resolved's
// drop-in, the ufw rules, docker.slice and docker.service's drop-in, then the units. A change
// restarts dockerd, which live-restore carries past without stopping the running containers. A
// new daemon.json applies to the networks and containers created afterwards, and to the default
// bridge only as dockerd starts with no container running.
func configuration(m effect.Machine, u effect.Units, out effect.Out, d values.Docker, size DiskSize) func(context.Context) ([]host.Resource, error) {
	return func(_ context.Context) ([]host.Resource, error) {
		d, err := WithLimits(m, d)
		if err != nil {
			return nil, err
		}
		cache, err := cacheLimits(m, d, size)
		if err != nil {
			return nil, err
		}
		daemonJSON, err := DaemonJSON(d, cache)
		if err != nil {
			return nil, err
		}
		slice, err := Slice(d)
		if err != nil {
			return nil, err
		}
		resolved, err := Resolved(d)
		if err != nil {
			return nil, err
		}
		override, err := Override()
		if err != nil {
			return nil, err
		}
		daemon := host.Validated{
			File: host.File{M: m, Path: "/etc/docker/daemon.json", Content: daemonJSON, Mode: 0o644},
			Check: func(path string) (string, []string) {
				return "dockerd", []string{"--validate", "--config-file", path}
			},
		}
		resolvedFile := host.File{M: m, Path: "/etc/systemd/resolved.conf.d/docker.conf", Content: resolved, Mode: 0o644}
		sliceFile := host.File{M: m, Path: "/etc/systemd/system/docker.slice", Content: slice, Mode: 0o644}
		overrideFile := host.File{M: m, Path: "/etc/systemd/system/docker.service.d/override.conf", Content: override, Mode: 0o644}
		return []host.Resource{
			daemon,
			host.Dir{M: m, Path: "/etc/systemd/resolved.conf.d", Mode: 0o755},
			resolvedFile,
			host.Unit{M: m, U: u, Out: out, Name: "systemd-resolved.service", Run: host.Running, Watch: []host.Resource{resolvedFile}},
			ufw.Rules{M: m, Out: out, Comment: resolverComment, Want: resolverRules(d), Owns: ownsResolverRule, NeedsSystemd: true},
			host.Dir{M: m, Path: "/etc/systemd/system/docker.service.d", Mode: 0o755},
			sliceFile,
			overrideFile,
			host.DaemonReload{M: m, U: u, Watch: []host.Resource{sliceFile, overrideFile}},
			host.Unit{M: m, U: u, Out: out, Name: "docker.service", Enabled: true, Run: host.Running, Watch: []host.Resource{daemon, overrideFile}},
		}, nil
	}
}
