package cli

import (
	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/values"
)

// serverConf is a server's name and its settings, or the refusal of its file.
type serverConf struct {
	Name string
	Conf values.Conf
	Err  error
}

// refusedReport is the report of a server whose file ws refuses: a check that holds the
// refusal and the way out, for the text and the JSON alike.
func refusedReport(sc serverConf) step.Report {
	return step.Report{Step: "values.yml", Items: []step.Item{{
		Kind: step.CheckItem, Name: sc.Name, Want: "a file ws accepts", Found: sc.Err.Error(),
		Conflict: values.BadFileHint(sc.Name),
	}}}
}

// stateConfs returns the servers that state reads. With several servers and no --server, that is each
// of them, and grouped is true: the report then shows each under its name. A server whose file is
// refused carries the refusal in Err. Otherwise it returns the one that loadMacConf picks.
func stateConfs(p enginecli.Env, server *string) (confs []serverConf, grouped bool, err error) {
	if *server == "" && p.Euid != 0 {
		mac := values.Path(p.Getenv, p.Pwd)
		names, err := values.Servers(p.Machine, mac)
		if err != nil {
			return nil, false, err
		}
		if len(names) > 1 {
			for _, n := range names {
				c, err := values.Load(p.Machine, mac, values.ServerPath(mac, n))
				confs = append(confs, serverConf{n, c, err})
			}
			return confs, true, nil
		}
	}
	name, settings, err := loadMacConf(p, server)
	return []serverConf{{name, settings, nil}}, false, err
}
