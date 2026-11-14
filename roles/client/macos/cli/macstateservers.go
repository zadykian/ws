package cli

import (
	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/roles/client/macos/conf"
)

// serverConf is a server's name and its settings.
type serverConf struct {
	Name string
	Conf conf.Conf
}

// stateConfs returns the servers that state reads. With several servers and no --server, that is each
// of them, and grouped is true: the report then shows each under its name. Else it returns the one
// that loadMacConf picks.
func stateConfs(p enginecli.Env, server *string) (confs []serverConf, grouped bool, err error) {
	if *server == "" && p.Euid != 0 {
		mac := conf.Path(p.Getenv, p.Pwd)
		names, err := conf.Servers(p.Machine, mac)
		if err != nil {
			return nil, false, err
		}
		if len(names) > 1 {
			for _, n := range names {
				c, err := conf.Load(p.Machine, mac, conf.ServerPath(mac, n))
				if err != nil {
					return nil, false, err
				}
				confs = append(confs, serverConf{n, c})
			}
			return confs, true, nil
		}
	}
	name, settings, err := loadMacConf(p, server)
	return []serverConf{{name, settings}}, false, err
}
