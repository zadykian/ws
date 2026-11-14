package cli

import (
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/internal/launchd"
	"github.com/zadykian/ws/roles/client/macos/tunnel/daemon/settings"
)

// daemonOf returns the daemon of --server, or the one installed: status runs under sudo, with no WS_CONF.
func daemonOf(m effect.Machine, server string) (launchd.Daemon, error) {
	if server != "" {
		if !conf.IsName(server) {
			return launchd.Daemon{}, fmt.Errorf("--server %s is not a server's name: %s", server, conf.NameForm)
		}
		return launchd.For(server), nil
	}
	names, err := launchd.Installed(m)
	switch {
	case err != nil:
		return launchd.Daemon{}, err
	case len(names) == 0:
		return launchd.Daemon{}, fmt.Errorf("no tunnel daemon in /Library/LaunchDaemons: run ws client macos apply")
	case len(names) > 1:
		return launchd.Daemon{}, fmt.Errorf("several tunnel daemons, %s: name one by --server", conf.List(names))
	}
	return launchd.For(names[0]), nil
}

// daemonPaths names the files of d. The key is the Mac's, whatever the server.
func daemonPaths(d launchd.Daemon) settings.Paths {
	return settings.Paths{Conf: d.Conf, Key: launchd.Dir + "/private.key", Routes: d.Routes, State: d.State, Log: d.Log,
		Root: launchd.Dir}
}
