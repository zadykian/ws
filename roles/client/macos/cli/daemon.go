package cli

import (
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/internal/launchd"
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
