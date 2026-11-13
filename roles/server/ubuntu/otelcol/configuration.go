package otelcol

import (
	"context"
	"errors"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/account"
	"github.com/zadykian/ws/engine/core/host/systemd"
)

const (
	dropIns   = "/etc/systemd/system/otelcol-contrib.service.d"
	configDir = "/etc/otelcol-contrib"
)

// remoteEndpoint reads the remote collector's endpoint from the user's ~/.secrets/env, or ""
// where the file or the variable is missing. A bad value stops the run.
func remoteEndpoint(m effect.Machine, user account.User) (string, error) {
	env, err := user.ReadHome(m, ".secrets/env")
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	endpoint := Endpoint(env)
	if endpoint == "" {
		return "", nil
	}
	return endpoint, CheckEndpoint(endpoint)
}

// configuration makes the parts that configure the collector and run it: the drop-ins, the
// endpoint for the service, the config, then the unit. The endpoint file is written before the
// config and removed after it, so that a run cut short never leaves a config that names a
// variable the service lacks. The collector checks the config before it replaces the running
// one's. The remote endpoint is a stand-in there, as its value is only in the service's
// environment. A change of any of them restarts the collector: a new endpoint needs a restart, as
// the collector's reload rereads its config alone.
func configuration(m effect.Machine, u effect.Units, out effect.Out, user account.User) func(context.Context) ([]host.Resource, error) {
	return func(context.Context) ([]host.Resource, error) {
		endpoint, err := remoteEndpoint(m, user)
		if err != nil {
			return nil, err
		}
		journal, err := files.ReadFile("files/journal.conf")
		if err != nil {
			return nil, err
		}
		claude, err := files.ReadFile("files/claude.conf")
		if err != nil {
			return nil, err
		}
		cfg, err := Config(endpoint)
		if err != nil {
			return nil, err
		}
		journalFile := host.File{M: m, Path: dropIns + "/journal.conf", Content: journal, Mode: 0o644}
		claudeFile := host.File{M: m, Path: dropIns + "/claude.conf", Content: claude, Mode: 0o644}
		config := host.Validated{
			File: host.File{M: m, Path: configDir + "/config.yaml", Content: cfg, Mode: 0o644},
			Check: func(path string) (string, []string) {
				return "env", []string{"OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT=http://remote.invalid:4317",
					"/usr/bin/otelcol-contrib", "validate", "--config=" + path}
			},
		}
		watch := []host.Resource{journalFile, claudeFile, config}
		parts := []host.Resource{host.Dir{M: m, Path: dropIns, Mode: 0o755}, journalFile, claudeFile}
		if endpoint != "" {
			env, err := Env(user.Home, endpoint)
			if err != nil {
				return nil, err
			}
			file := host.File{M: m, Path: configDir + "/claude.env", Content: env, Mode: 0o600, Secret: true}
			parts = append(parts, file, config)
			watch = append(watch, file)
		} else {
			gone := host.Absent{M: m, Path: configDir + "/claude.env"}
			parts = append(parts, config, gone)
			watch = append(watch, gone)
		}
		parts = append(parts,
			systemd.DaemonReload{M: m, U: u, Watch: []host.Resource{journalFile, claudeFile}},
			systemd.Unit{M: m, U: u, Out: out, Name: "otelcol-contrib.service", Enabled: true, Run: systemd.Running, Watch: watch},
		)
		return parts, nil
	}
}
