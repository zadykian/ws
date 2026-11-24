package cli

import (
	"fmt"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/internal/setup"
)

// UpdateSetups is the setup that ws update ends in on Ubuntu: that of the env in the miserc, which
// a setup wrote. A machine with no miserc has run none, so the update runs no setup there.
func UpdateSetups() enginecli.Setups {
	return enginecli.Setups{
		Names: []string{"ws server setup"},
		Pick: func(m effect.Machine) ([]string, error) {
			envs, err := setup.SideEnvs(m)
			if err != nil || len(envs) == 0 {
				return nil, err
			}
			if len(envs) != 1 || envs[0] != "server" {
				return nil, fmt.Errorf("%s names the envs %v, and ws update knows server alone", setup.MisercPath, envs)
			}
			return []string{"server", "setup"}, nil
		},
	}
}
