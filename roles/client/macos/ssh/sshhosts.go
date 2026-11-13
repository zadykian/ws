package ssh

import (
	"fmt"

	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// hosts returns a block's inputs for each server that mac.conf knows. The run's own come from its
// conf, every other's from its server.conf. A file that fails to load stops the step.
func (s SSH) hosts() ([]sshcfg.Host, error) {
	names, err := conf.Known(s.M, s.MacConf, s.Server)
	if err != nil {
		return nil, err
	}
	hosts := make([]sshcfg.Host, 0, len(names))
	for _, name := range names {
		c := s.Conf
		if name != s.Server {
			if c, err = conf.Load(s.M, s.MacConf, conf.ServerPath(s.MacConf, name)); err != nil {
				return nil, fmt.Errorf("server %s: %w", name, err)
			}
		}
		hosts = append(hosts, sshcfg.Host{Name: name, Addr: c.TunnelSlot().Server(), User: c.User})
	}
	return hosts, nil
}
