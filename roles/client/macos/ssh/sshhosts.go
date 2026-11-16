package ssh

import (
	"fmt"

	"github.com/zadykian/ws/roles/client/macos/values"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// hosts returns a block's inputs for each server that values.yml knows. The run's own come from its
// conf, every other's from its values.yml. A file that fails to load stops the step.
func (s SSH) hosts() ([]sshcfg.Host, error) {
	names, err := values.Known(s.M, s.MacConf, s.Server)
	if err != nil {
		return nil, err
	}
	hosts := make([]sshcfg.Host, 0, len(names))
	for _, name := range names {
		c := s.Conf
		if name != s.Server {
			if c, err = values.Load(s.M, s.MacConf, values.ServerPath(s.MacConf, name)); err != nil {
				return nil, fmt.Errorf("server %s: %w", name, err)
			}
		}
		hosts = append(hosts, sshcfg.Host{Name: name, Addr: c.TunnelSlot().Server(), User: c.User})
	}
	return hosts, nil
}
