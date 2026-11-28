package cli

import (
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/macos/roles/tunnel"
	"github.com/zadykian/ws/macos/roles/tunnel/daemon/settings"
	"github.com/zadykian/ws/macos/values"
)

// daemonOf returns the daemon of --server, or the one installed: status runs under sudo, with no WS_VALUES.
func daemonOf(m effect.Machine, server string) (tunnel.Daemon, error) {
	if server != "" {
		if !values.IsName(server) {
			return tunnel.Daemon{}, fmt.Errorf("--server %s is not a server's name: %s", server, values.NameForm)
		}
		return tunnel.For(server), nil
	}
	names, err := tunnel.Installed(m)
	switch {
	case err != nil:
		return tunnel.Daemon{}, err
	case len(names) == 0:
		return tunnel.Daemon{}, fmt.Errorf("no tunnel daemon in /Library/LaunchDaemons: run mise -C /opt/ws/macos bootstrap")
	case len(names) > 1:
		return tunnel.Daemon{}, fmt.Errorf("several tunnel daemons, %s: name one by --server", values.List(names))
	}
	return tunnel.For(names[0]), nil
}

// daemonPaths names the files of d. The key and the yield are the Mac's, whatever the server.
func daemonPaths(d tunnel.Daemon) settings.Paths {
	return settings.Paths{Conf: d.Conf, Key: tunnel.Dir + "/private.key", Routes: d.Routes, State: d.State, Log: d.Log,
		Root: tunnel.Dir, Yield: tunnel.YieldFile}
}
